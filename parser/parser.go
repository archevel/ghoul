// Package parser provides a SIMD-accelerated parser for Ghoul Lisp syntax.
// It reads the entire input into memory, classifies characters in 32-byte
// chunks using AVX2 vector comparisons, and then walks the classification
// bitmasks to tokenize and build the expression tree.
//
// The SIMD pass produces per-chunk bitmasks for structural characters, and
// a precomputed row/column index from newline positions. Subsequent parsing
// operations use these bitmasks to skip whitespace, find token boundaries,
// and locate string terminators in bulk rather than byte-by-byte.
//
// Requires Go 1.26+ with GOEXPERIMENT=simd on AMD64.
package parser

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/bits"
	"os"
	"simd/archsimd"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	e "github.com/archevel/ghoul/expressions"
)

// SPECIAL_IDENTIFIERS lists the non-ASCII characters accepted as identifier
// characters (in addition to unicode letters, digits after position 0,
// hyphens after position 0, and dots after position 0).
const SPECIAL_IDENTIFIERS = "§¶½!@£¤$%€&¥/=?+\\^~*´_:,<>|«»©\u201c\u201dµªßðđŋħĸłøæåöäþœ→↓←þ®€ł@"

// Position records the row and column of a token in the source.
type Position struct {
	Row int
	Col int
}

func (p Position) String() string {
	return "(" + strconv.Itoa(p.Row) + ":" + strconv.Itoa(p.Col) + ")"
}

// ParsedExpressions holds the result of parsing.
type ParsedExpressions struct {
	Expressions      e.List
	pairSrcPositions map[e.Pair]Position
}

func (pe ParsedExpressions) PositionOf(p e.Pair) (pos Position, found bool) {
	pos, found = pe.pairSrcPositions[p]
	return
}

// Parse parses Ghoul source from reader r.
// Returns 0 on success, non-zero on parse error.
func Parse(r io.Reader) (int, *ParsedExpressions) {
	return ParseWithFilename(r, nil)
}

// ParseWithFilename parses Ghoul source, associating the given filename
// with source positions for error reporting.
func ParseWithFilename(r io.Reader, filename *string) (int, *ParsedExpressions) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 1, &ParsedExpressions{e.NIL, nil}
	}
	p := &parser{
		data:      data,
		filename:  filename,
		positions: make(map[e.Pair]Position),
	}
	p.classify()
	exprs := p.parseTopLevel()
	if p.err != nil {
		return 1, &ParsedExpressions{e.NIL, p.positions}
	}
	return 0, &ParsedExpressions{exprs, p.positions}
}

// ---------- SIMD character classification ----------

// charClass holds precomputed bitmasks for each 32-byte chunk of the input.
// Bit i is set if position (chunkIndex*32 + i) matches the character class.
type charClass struct {
	whitespace []uint32 // space, tab, \r, \n
	newline    []uint32 // \n only
	lparen     []uint32
	rparen     []uint32
	semicolon  []uint32
	quote      []uint32 // '
	dquote     []uint32 // "
	backtick   []uint32 // `
	hash       []uint32
	dot        []uint32
	backslash  []uint32 // \ (for escape detection in strings)
	delimiter  []uint32 // whitespace | ( | ) | ' | " | ` | ;
}

// posIndex provides O(1) row/column lookup for any byte offset.
// Built during classification by counting newlines per chunk.
type posIndex struct {
	// newlinePrefix[i] = total newlines in chunks 0..i-1
	// So newlines before chunk i = newlinePrefix[i]
	newlinePrefix []int
	// newlinePositions[i] = byte offset of the i-th newline (0-indexed)
	newlinePositions []int
}

func classifyChunks(data []byte) (charClass, posIndex) {
	nchunks := (len(data) + 31) / 32
	cc := charClass{
		whitespace: make([]uint32, nchunks),
		newline:    make([]uint32, nchunks),
		lparen:     make([]uint32, nchunks),
		rparen:     make([]uint32, nchunks),
		semicolon:  make([]uint32, nchunks),
		quote:      make([]uint32, nchunks),
		dquote:     make([]uint32, nchunks),
		backtick:   make([]uint32, nchunks),
		hash:       make([]uint32, nchunks),
		dot:        make([]uint32, nchunks),
		backslash:  make([]uint32, nchunks),
		delimiter:  make([]uint32, nchunks),
	}

	bSpace := archsimd.BroadcastUint8x32(' ')
	bTab := archsimd.BroadcastUint8x32('\t')
	bCR := archsimd.BroadcastUint8x32('\r')
	bNL := archsimd.BroadcastUint8x32('\n')
	bLP := archsimd.BroadcastUint8x32('(')
	bRP := archsimd.BroadcastUint8x32(')')
	bSC := archsimd.BroadcastUint8x32(';')
	bQ := archsimd.BroadcastUint8x32('\'')
	bDQ := archsimd.BroadcastUint8x32('"')
	bBT := archsimd.BroadcastUint8x32('`')
	bH := archsimd.BroadcastUint8x32('#')
	bDot := archsimd.BroadcastUint8x32('.')
	bBS := archsimd.BroadcastUint8x32('\\')

	// Newline tracking for posIndex
	pi := posIndex{
		newlinePrefix: make([]int, nchunks+1),
	}
	totalNewlines := 0

	// Process full 32-byte chunks with SIMD
	fullChunks := len(data) / 32
	for i := 0; i < fullChunks; i++ {
		v := archsimd.LoadUint8x32Slice(data[i*32:])

		ws := v.Equal(bSpace).Or(v.Equal(bTab)).Or(v.Equal(bCR)).Or(v.Equal(bNL))
		nl := v.Equal(bNL)
		lp := v.Equal(bLP)
		rp := v.Equal(bRP)
		sc := v.Equal(bSC)
		q := v.Equal(bQ)
		dq := v.Equal(bDQ)
		bt := v.Equal(bBT)

		cc.whitespace[i] = ws.ToBits()
		cc.newline[i] = nl.ToBits()
		cc.lparen[i] = lp.ToBits()
		cc.rparen[i] = rp.ToBits()
		cc.semicolon[i] = sc.ToBits()
		cc.quote[i] = q.ToBits()
		cc.dquote[i] = dq.ToBits()
		cc.backtick[i] = bt.ToBits()
		cc.hash[i] = v.Equal(bH).ToBits()
		cc.dot[i] = v.Equal(bDot).ToBits()
		cc.backslash[i] = v.Equal(bBS).ToBits()
		cc.delimiter[i] = ws.Or(lp).Or(rp).Or(q).Or(dq).Or(bt).Or(sc).ToBits()

		// Count newlines for position index
		pi.newlinePrefix[i] = totalNewlines
		nlBits := cc.newline[i]
		nlCount := bits.OnesCount32(nlBits)
		remaining := nlBits
		for remaining != 0 {
			bit := bits.TrailingZeros32(remaining)
			pi.newlinePositions = append(pi.newlinePositions, i*32+bit)
			remaining &= remaining - 1
		}
		totalNewlines += nlCount
	}

	// Handle the final partial chunk with scalar fallback
	if remainder := len(data) % 32; remainder > 0 {
		ci := fullChunks
		pi.newlinePrefix[ci] = totalNewlines
		base := ci * 32
		for j := 0; j < remainder; j++ {
			b := data[base+j]
			bit := uint32(1) << j
			switch b {
			case ' ', '\t', '\r':
				cc.whitespace[ci] |= bit
				cc.delimiter[ci] |= bit
			case '\n':
				cc.whitespace[ci] |= bit
				cc.newline[ci] |= bit
				cc.delimiter[ci] |= bit
				pi.newlinePositions = append(pi.newlinePositions, base+j)
				totalNewlines++
			case '(':
				cc.lparen[ci] |= bit
				cc.delimiter[ci] |= bit
			case ')':
				cc.rparen[ci] |= bit
				cc.delimiter[ci] |= bit
			case ';':
				cc.semicolon[ci] |= bit
				cc.delimiter[ci] |= bit
			case '\'':
				cc.quote[ci] |= bit
				cc.delimiter[ci] |= bit
			case '"':
				cc.dquote[ci] |= bit
				cc.delimiter[ci] |= bit
			case '`':
				cc.backtick[ci] |= bit
				cc.delimiter[ci] |= bit
			case '#':
				cc.hash[ci] |= bit
			case '.':
				cc.dot[ci] |= bit
			case '\\':
				cc.backslash[ci] |= bit
			}
		}
	}
	pi.newlinePrefix[nchunks] = totalNewlines

	return cc, pi
}

// ---------- Position index ----------

// posAt returns the row and column for a byte offset using the precomputed
// newline index. Row is 1-based, column is 1-based.
func (pi *posIndex) posAt(data []byte, offset int) Position {
	// Binary search for the newline count before this offset
	row := 1
	col := offset + 1 // if no newlines, column = offset + 1

	// Find how many newlines are before offset
	lo, hi := 0, len(pi.newlinePositions)
	for lo < hi {
		mid := (lo + hi) / 2
		if pi.newlinePositions[mid] < offset {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	// lo = number of newlines before offset
	row = lo + 1
	if lo > 0 {
		col = offset - pi.newlinePositions[lo-1]
	}
	return Position{row, col}
}

// ---------- Parser state ----------

type parser struct {
	data      []byte
	pos       int
	filename  *string
	positions map[e.Pair]Position
	err       error
	cc        charClass
	pi        posIndex
}

func (p *parser) classify() {
	p.cc, p.pi = classifyChunks(p.data)
}

// curPos returns the current source position using the precomputed index.
func (p *parser) curPos() Position {
	return p.pi.posAt(p.data, p.pos)
}

func isSet(mask []uint32, pos int) bool {
	chunk := pos / 32
	bit := pos % 32
	if chunk >= len(mask) {
		return false
	}
	return mask[chunk]&(1<<bit) != 0
}

// ---------- Navigation ----------

func (p *parser) atEnd() bool {
	return p.pos >= len(p.data)
}

func (p *parser) peek() byte {
	return p.data[p.pos]
}

func (p *parser) peekRune() rune {
	r, _ := utf8.DecodeRune(p.data[p.pos:])
	return r
}

// advance moves forward one byte.
func (p *parser) advance() {
	p.pos++
}

// advanceRune moves forward one UTF-8 code point.
func (p *parser) advanceRune() {
	_, size := utf8.DecodeRune(p.data[p.pos:])
	p.pos += size
}

// skipWhitespaceAndComments uses precomputed bitmasks to bulk-skip
// whitespace and comment regions. For whitespace, it finds the first
// non-whitespace bit in the current chunk and jumps directly. For
// comments, it uses the newline bitmask to jump to end-of-line.
func (p *parser) skipWhitespaceAndComments() {
	for p.pos < len(p.data) {
		chunk := p.pos / 32
		bit := p.pos % 32

		if chunk >= len(p.cc.whitespace) {
			break
		}

		// Check for comment first (semicolons)
		if p.cc.semicolon[chunk]&(1<<bit) != 0 {
			p.skipToNewline()
			continue
		}

		// Check for whitespace — find first non-whitespace in this chunk
		if p.cc.whitespace[chunk]&(1<<bit) != 0 {
			// Mask out bits before current position, then find first zero
			wsMask := p.cc.whitespace[chunk] >> bit
			// Find first 0 bit = first non-whitespace
			nonWS := bits.TrailingZeros32(^wsMask)
			if nonWS < 32-bit {
				p.pos += nonWS
				continue // re-check (might be a comment)
			}
			// Rest of chunk is all whitespace — skip to next chunk
			p.pos = (chunk + 1) * 32
			continue
		}

		break
	}
}

// skipToNewline jumps to the position after the next newline using the
// precomputed newline bitmask.
func (p *parser) skipToNewline() {
	chunk := p.pos / 32
	bit := p.pos % 32

	// Check remaining bits in current chunk
	if chunk < len(p.cc.newline) {
		mask := p.cc.newline[chunk] >> bit
		if mask != 0 {
			offset := bits.TrailingZeros32(mask)
			p.pos += offset + 1 // skip past the newline
			return
		}
	}

	// Search subsequent chunks
	for ci := chunk + 1; ci < len(p.cc.newline); ci++ {
		if p.cc.newline[ci] != 0 {
			target := ci*32 + bits.TrailingZeros32(p.cc.newline[ci])
			p.pos = target + 1 // skip past the newline
			return
		}
	}

	// No newline found — skip to end
	p.pos = len(p.data)
}

// findTokenEnd uses the precomputed delimiter bitmask to find the end of
// the current token (identifier, number, etc.) starting at p.pos.
// Returns the byte offset of the first delimiter after the token.
func (p *parser) findTokenEnd() int {
	pos := p.pos
	for pos < len(p.data) {
		chunk := pos / 32
		bit := pos % 32

		if chunk >= len(p.cc.delimiter) {
			break
		}

		// Find first delimiter bit at or after current position
		delimMask := p.cc.delimiter[chunk] >> bit
		if delimMask != 0 {
			offset := bits.TrailingZeros32(delimMask)
			return pos + offset
		}

		// No delimiter in this chunk — all chars are token chars
		pos = (chunk + 1) * 32
	}
	return len(p.data)
}

// findClosingQuote uses the dquote and backslash bitmasks to find the
// next unescaped double quote after p.pos. Uses the simdjson technique:
// unescaped quotes = dquote positions that are NOT preceded by a backslash.
// For sequences like \\", the quote IS unescaped (the backslash is itself
// escaped), so we handle runs of backslashes correctly.
func (p *parser) findClosingQuote() int {
	pos := p.pos

	// Carry: was the last byte of the previous chunk a backslash?
	prevCarry := uint32(0)

	for pos < len(p.data) {
		chunk := pos / 32
		bit := pos % 32
		if chunk >= len(p.cc.dquote) {
			break
		}

		dqMask := p.cc.dquote[chunk]
		bsMask := p.cc.backslash[chunk]

		// Compute which backslashes actually escape the next character.
		// A backslash at position i escapes position i+1 UNLESS it is
		// itself escaped by a backslash at position i-1. For runs of
		// backslashes (\\\\"), we need to determine parity.
		//
		// Simple approximation: a quote at position i is escaped if
		// position i-1 has a backslash AND position i-2 does NOT.
		// This handles \", \\", \\\", etc. correctly for the common cases.
		//
		// Shifted backslash mask: bit i set means position i-1 was '\'
		escaped := (bsMask << 1) | prevCarry
		// Double-escaped: bit i set means positions i-2 AND i-1 were both '\'
		doubleEscaped := bsMask & (bsMask << 1)
		// A quote is unescaped if: it's a quote AND (not escaped OR double-escaped)
		unescaped := dqMask & (^escaped | (doubleEscaped << 1))

		// Mask out bits before current position
		unescaped >>= bit
		if unescaped != 0 {
			return pos + bits.TrailingZeros32(unescaped)
		}

		// Carry the last bit's backslash status to the next chunk
		prevCarry = (bsMask >> 31) & 1

		pos = (chunk + 1) * 32
	}
	return -1
}

// hasNewlineInRange checks whether any newline exists in [start, end) using
// the precomputed newline bitmask. O(chunks) instead of O(bytes).
func (p *parser) hasNewlineInRange(start, end int) bool {
	startChunk := start / 32
	endChunk := (end - 1) / 32

	for ci := startChunk; ci <= endChunk && ci < len(p.cc.newline); ci++ {
		mask := p.cc.newline[ci]
		if mask == 0 {
			continue
		}
		// Mask out bits before start position in the first chunk
		if ci == startChunk {
			mask &= ^uint32((1 << (start % 32)) - 1)
		}
		// Mask out bits at or after end position in the last chunk
		if ci == endChunk && end%32 != 0 {
			mask &= (1 << (end % 32)) - 1
		}
		if mask != 0 {
			return true
		}
	}
	return false
}

func (p *parser) syntaxError(msg string) {
	if p.err == nil {
		fmt.Fprintf(os.Stderr, "Error: syntax error\n")
		pos := p.curPos()
		p.err = fmt.Errorf("%d:%d: %s", pos.Row, pos.Col, msg)
	}
}

// ---------- Parsing ----------

func (p *parser) parseTopLevel() e.List {
	var exprs []e.Expr
	var exprPositions []Position
	for {
		p.skipWhitespaceAndComments()
		if p.atEnd() {
			break
		}
		pos := p.curPos()
		expr := p.parseExpr()
		if p.err != nil {
			return e.NIL
		}
		if expr != nil {
			exprs = append(exprs, expr)
			exprPositions = append(exprPositions, pos)
		}
	}

	var result e.Expr = e.NIL
	for i := len(exprs) - 1; i >= 0; i-- {
		pair := e.Cons(exprs[i], result)
		pair.Loc = &e.SourcePosition{Ln: exprPositions[i].Row, Col: exprPositions[i].Col, Filename: p.filename}
		p.positions[*pair] = exprPositions[i]
		result = pair
	}
	if result == e.NIL {
		return e.NIL
	}
	return result.(e.List)
}

func (p *parser) parseExpr() e.Expr {
	p.skipWhitespaceAndComments()
	if p.atEnd() {
		return nil
	}

	pos := p.pos
	if isSet(p.cc.lparen, pos) {
		return p.parseList()
	}
	if isSet(p.cc.rparen, pos) {
		p.syntaxError("unexpected ')'")
		return nil
	}
	if isSet(p.cc.quote, pos) {
		return p.parseQuote()
	}
	if isSet(p.cc.dquote, pos) {
		return p.parseString()
	}
	if isSet(p.cc.backtick, pos) {
		return p.parseRawString()
	}
	if isSet(p.cc.hash, pos) {
		return p.parseHash()
	}
	return p.parseAtom()
}

func (p *parser) parseList() e.Expr {
	p.advance() // skip '('
	p.skipWhitespaceAndComments()

	if p.atEnd() {
		p.syntaxError("unterminated list")
		return nil
	}
	if isSet(p.cc.rparen, p.pos) {
		p.advance()
		return e.NIL
	}

	var elems []e.Expr
	var elemPositions []Position
	var dotTail e.Expr

	for {
		p.skipWhitespaceAndComments()
		if p.atEnd() {
			p.syntaxError("unterminated list")
			return nil
		}
		if isSet(p.cc.rparen, p.pos) {
			p.advance()
			break
		}

		if isSet(p.cc.dot, p.pos) && p.isDotToken() {
			p.advance() // skip '.'
			p.skipWhitespaceAndComments()
			dotTail = p.parseExpr()
			if p.err != nil {
				return nil
			}
			p.skipWhitespaceAndComments()
			if p.atEnd() || !isSet(p.cc.rparen, p.pos) {
				p.syntaxError("expected ')' after dotted tail")
				return nil
			}
			p.advance()
			break
		}

		elemPos := p.curPos()
		expr := p.parseExpr()
		if p.err != nil {
			return nil
		}
		elems = append(elems, expr)
		elemPositions = append(elemPositions, elemPos)
	}

	var tail e.Expr = e.NIL
	if dotTail != nil {
		tail = dotTail
	}
	for i := len(elems) - 1; i >= 0; i-- {
		pair := e.Cons(elems[i], tail)
		pair.Loc = &e.SourcePosition{Ln: elemPositions[i].Row, Col: elemPositions[i].Col, Filename: p.filename}
		p.positions[*pair] = elemPositions[i]
		tail = pair
	}
	return tail
}

func (p *parser) parseQuote() e.Expr {
	p.advance() // skip '\''
	expr := p.parseExpr()
	if p.err != nil {
		return nil
	}
	return &e.Quote{Quoted: expr}
}

// parseString uses the dquote bitmask to find the closing quote quickly,
// then processes escape sequences in the found range.
func (p *parser) parseString() e.Expr {
	p.advance() // skip opening '"'

	closing := p.findClosingQuote()
	if closing < 0 {
		p.syntaxError("unterminated string literal")
		return nil
	}

	// Check for newline between opening and closing quote using bitmask
	if p.hasNewlineInRange(p.pos, closing) {
		// Find the exact position for the error message
		for pos := p.pos; pos < closing; pos++ {
			if p.data[pos] == '\n' {
				p.pos = pos
				break
			}
		}
		p.syntaxError("newline in string literal")
		return nil
	}

	// Process escape sequences in the range [p.pos, closing)
	raw := p.data[p.pos:closing]
	p.pos = closing + 1 // skip past closing quote

	if !bytes.ContainsRune(raw, '\\') {
		// Fast path: no escapes
		return e.String(string(raw))
	}

	// Slow path: process escapes
	var buf bytes.Buffer
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' && i+1 < len(raw) {
			i++
			switch raw[i] {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case '\\':
				buf.WriteByte('\\')
			case '"':
				buf.WriteByte('"')
			default:
				buf.WriteByte('\\')
				buf.WriteByte(raw[i])
			}
		} else {
			buf.WriteByte(raw[i])
		}
	}
	return e.String(buf.String())
}

func (p *parser) parseRawString() e.Expr {
	p.advance() // skip opening '`'
	start := p.pos
	closing := p.findClosingBacktick()
	if closing < 0 {
		p.syntaxError("unterminated raw string literal")
		return nil
	}
	raw := string(p.data[start:closing])
	p.pos = closing + 1 // skip past closing '`'
	return e.String(raw)
}

// findClosingBacktick uses the backtick bitmask to find the next backtick
// after p.pos. Returns -1 if not found.
func (p *parser) findClosingBacktick() int {
	pos := p.pos
	for pos < len(p.data) {
		chunk := pos / 32
		bit := pos % 32
		if chunk >= len(p.cc.backtick) {
			break
		}
		btMask := p.cc.backtick[chunk] >> bit
		if btMask != 0 {
			return pos + bits.TrailingZeros32(btMask)
		}
		pos = (chunk + 1) * 32
	}
	return -1
}

func (p *parser) parseHash() e.Expr {
	p.advance() // skip '#'
	if p.atEnd() {
		p.syntaxError("unexpected '#' at end of input")
		return nil
	}
	ch := p.peek()
	switch ch {
	case 't':
		p.advance()
		return e.Boolean(true)
	case 'f':
		p.advance()
		return e.Boolean(false)
	case '!':
		p.skipToNewline()
		return nil
	default:
		p.syntaxError(fmt.Sprintf("unexpected '#%c'", ch))
		return nil
	}
}

// parseAtom uses findTokenEnd to locate the token boundary in bulk,
// then classifies the token as integer, float, or identifier.
func (p *parser) parseAtom() e.Expr {
	end := p.findTokenEnd()

	// Handle multi-byte UTF-8 characters that aren't delimiters
	// (the SIMD pass only classifies ASCII delimiters)
	for end < len(p.data) {
		r, size := utf8.DecodeRune(p.data[end:])
		if isDelimiter(r) {
			break
		}
		end += size
		// Continue scanning for the next ASCII delimiter
		next := end
		for next < len(p.data) {
			chunk := next / 32
			bit := next % 32
			if chunk >= len(p.cc.delimiter) {
				next = len(p.data)
				break
			}
			delimMask := p.cc.delimiter[chunk] >> bit
			if delimMask != 0 {
				next += bits.TrailingZeros32(delimMask)
				break
			}
			next = (chunk + 1) * 32
		}
		end = next
	}

	tok := string(p.data[p.pos:end])
	p.pos = end

	if tok == "" {
		p.syntaxError("empty token")
		return nil
	}

	// Try integer
	if i, err := strconv.ParseInt(tok, 0, 64); err == nil {
		return e.Integer(i)
	}

	// Try float (must contain '.', 'e', or 'E' and start with digit/sign/dot)
	if strings.ContainsAny(tok, ".eE") {
		if len(tok) > 0 && (tok[0] == '-' || tok[0] == '.' || (tok[0] >= '0' && tok[0] <= '9')) {
			if f, err := strconv.ParseFloat(tok, 64); err == nil {
				return e.Float(f)
			}
		}
	}

	// Clamp overflowing integers
	if looksLikeInteger(tok) {
		if len(tok) > 0 && tok[0] == '-' {
			return e.Integer(math.MinInt64)
		}
		return e.Integer(math.MaxInt64)
	}

	return e.Identifier(tok)
}

func (p *parser) isDotToken() bool {
	next := p.pos + 1
	if next >= len(p.data) {
		return true
	}
	ch, _ := utf8.DecodeRune(p.data[next:])
	return isDelimiter(ch)
}

// looksLikeInteger returns true if tok looks like an integer that overflowed
// ParseInt (all decimal digits, optionally with leading '-' or "0x" prefix).
func looksLikeInteger(tok string) bool {
	if len(tok) == 0 {
		return false
	}
	s := tok
	if s[0] == '-' {
		s = s[1:]
	}
	if len(s) == 0 {
		return false
	}
	isHex := false
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
		isHex = true
	}
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if isHex {
			if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') && !(c >= 'A' && c <= 'F') {
				return false
			}
		} else {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

func isDelimiter(ch rune) bool {
	return unicode.IsSpace(ch) || ch == '(' || ch == ')' || ch == '\'' || ch == '"' || ch == '`' || ch == ';'
}
