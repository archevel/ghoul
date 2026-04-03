package exhumer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	sc "text/scanner"
	"unicode"

	e "github.com/archevel/ghoul/bones"
)

var ErrNewLineInString = errors.New("New line in string")
var ErrEOFInString = errors.New("eof in string")

type Position struct {
	Row int
	Col int
}

func (p Position) String() string {
	return "(" + strconv.Itoa(p.Row) + ":" + strconv.Itoa(p.Col) + ")"
}

type schemeLexer struct {
	scanner   *bufio.Scanner
	result    *e.Node
	pos       *Position
	Filename  *string
	lastToken string
	lastTok   int
	Debug     bool
}

func scanToNextNonComment(scanner *bufio.Scanner) {
	for scanner.Scan() {
		data := scanner.Bytes()
		if len(data) > 0 && data[0] != ';' {
			return
		}
	}
}

func isIdentRune(ch rune, i int) bool {
	return strings.ContainsRune(SPECIAL_IDENTIFIERS, ch) || unicode.IsLetter(ch) || ((unicode.IsDigit(ch) || ch == '-' || ch == '.') && i > 0)
}

func (l *schemeLexer) Lex(lval *yySymType) int {

	scanToNextNonComment(l.scanner)

	if l.scanner.Err() != nil {
		lval.tok = l.scanner.Text()
		lval.col = l.pos.Col
		lval.row = l.pos.Row
		l.lastToken = lval.tok
		l.lastTok = UNEXPECTED_TOKEN
		if l.Debug {
			fmt.Fprintf(os.Stderr, "[LEX] scanner error at %d:%d, tok=%q\n", l.pos.Row, l.pos.Col, lval.tok)
		}
		return UNEXPECTED_TOKEN
	}

	data := l.scanner.Bytes()
	lval.col = l.pos.Col
	lval.row = l.pos.Row

	l.pos.Col += len(data)
	if len(data) > 0 {
		first := data[0]
		var tok int
		switch first {
		case '`':
			lastNewLine := bytes.LastIndexByte(data, '\n')

			if lastNewLine >= 0 {
				l.pos.Col = (len(data) - lastNewLine)
			}

			l.pos.Row += bytes.Count(data, []byte{'\n'})
			fallthrough
		case '"':
			str, _ := strconv.Unquote(l.scanner.Text())
			lval.tok = str
			l.lastToken = l.scanner.Text()
			l.lastTok = STRING
			if l.Debug {
				fmt.Fprintf(os.Stderr, "[LEX] %d:%d STRING %q\n", lval.row, lval.col, lval.tok)
			}
			return STRING
		case '(':
			l.lastToken = "("
			l.lastTok = BEG_LIST
			if l.Debug {
				fmt.Fprintf(os.Stderr, "[LEX] %d:%d BEG_LIST\n", lval.row, lval.col)
			}
			return BEG_LIST
		case ')':
			l.lastToken = ")"
			l.lastTok = END_LIST
			if l.Debug {
				fmt.Fprintf(os.Stderr, "[LEX] %d:%d END_LIST\n", lval.row, lval.col)
			}
			return END_LIST
		case '\'':
			l.lastToken = "'"
			l.lastTok = QUOTE
			if l.Debug {
				fmt.Fprintf(os.Stderr, "[LEX] %d:%d QUOTE\n", lval.row, lval.col)
			}
			return QUOTE
		case '.':
			if len(data) == 1 {
				l.lastToken = "."
				l.lastTok = DOT
				if l.Debug {
					fmt.Fprintf(os.Stderr, "[LEX] %d:%d DOT\n", lval.row, lval.col)
				}
				return DOT
			} else {
				lval.tok = l.scanner.Text()
				l.lastToken = lval.tok
				l.lastTok = IDENTIFIER
				if l.Debug {
					fmt.Fprintf(os.Stderr, "[LEX] %d:%d IDENTIFIER %q\n", lval.row, lval.col, lval.tok)
				}
				return IDENTIFIER
			}
		case '#':
			if len(data) > 1 {
				second := data[1]
				if second == 't' {
					l.lastToken = "#t"
					l.lastTok = TRUE
					if l.Debug {
						fmt.Fprintf(os.Stderr, "[LEX] %d:%d TRUE\n", lval.row, lval.col)
					}
					return TRUE
				} else if second == 'f' {
					l.lastToken = "#f"
					l.lastTok = FALSE
					if l.Debug {
						fmt.Fprintf(os.Stderr, "[LEX] %d:%d FALSE\n", lval.row, lval.col)
					}
					return FALSE
				} else if second == '!' {
					l.lastToken = "#!"
					l.lastTok = HASHBANG
					if l.Debug {
						fmt.Fprintf(os.Stderr, "[LEX] %d:%d HASHBANG\n", lval.row, lval.col)
					}
					return HASHBANG
				}
			} else {
				lval.tok = l.scanner.Text()
				l.lastToken = lval.tok
				l.lastTok = UNEXPECTED_TOKEN
				if l.Debug {
					fmt.Fprintf(os.Stderr, "[LEX] %d:%d UNEXPECTED_TOKEN %q\n", lval.row, lval.col, lval.tok)
				}
				return UNEXPECTED_TOKEN
			}
		default:
			tok = handleValue(lval, data)
			l.lastToken = lval.tok
			l.lastTok = tok
			if l.Debug {
				fmt.Fprintf(os.Stderr, "[LEX] %d:%d token=%d %q\n", lval.row, lval.col, tok, lval.tok)
			}
			return tok

		}
		lval.tok = l.scanner.Text()
		l.lastToken = lval.tok
		l.lastTok = UNEXPECTED_TOKEN
		if l.Debug {
			fmt.Fprintf(os.Stderr, "[LEX] %d:%d UNEXPECTED_TOKEN (fallthrough) %q\n", lval.row, lval.col, lval.tok)
		}
		return UNEXPECTED_TOKEN
	}

	l.lastToken = ""
	l.lastTok = 0
	if l.Debug {
		fmt.Fprintf(os.Stderr, "[LEX] %d:%d EOF\n", lval.row, lval.col)
	}
	return 0

}

func handleValue(lval *yySymType, data []byte) int {
	var subscanner sc.Scanner
	subscanner.IsIdentRune = isIdentRune
	subscanner.Mode = sc.ScanIdents | sc.ScanFloats
	subscanner.Init(bytes.NewReader(data))
	switch subscanner.Scan() {
	case '-':
		res, str := handleNeg(&subscanner)
		lval.tok = str
		return res
	case sc.Ident:
		lval.tok = subscanner.TokenText()
		return IDENTIFIER
	case sc.Int:
		lval.tok = subscanner.TokenText()
		return INTEGER
	case sc.Float:
		lval.tok = subscanner.TokenText()
		return FLOAT
	}

	lval.tok = subscanner.TokenText()
	return UNEXPECTED_TOKEN

}

func handleNeg(subscanner *sc.Scanner) (int, string) {
	var buffer bytes.Buffer
	buffer.WriteRune('-')
	pos := 1
	for {
		tok := subscanner.Scan()

		if tok == sc.EOF {
			return IDENTIFIER, buffer.String()
		}

		buffer.WriteString(subscanner.TokenText())
		switch tok {
		case '-':
			pos += 1

		case sc.Int:
			if pos == 1 {
				return INTEGER, buffer.String()
			} else {
				return IDENTIFIER, buffer.String()
			}
		case sc.Float:
			return FLOAT, buffer.String()
		case sc.Ident:
			return IDENTIFIER, buffer.String()
		default:
			return UNEXPECTED_TOKEN, buffer.String()
		}
	}
}

func (l *schemeLexer) Error(e string) {
	filename := "<stdin>"
	if l.Filename != nil {
		filename = *l.Filename
	}
	fmt.Fprintf(os.Stderr, "Error at %s:%d:%d: %s (last token: %q, tok type: %d)\n",
		filename, l.pos.Row, l.pos.Col, e, l.lastToken, l.lastTok)
}

const SPECIAL_IDENTIFIERS = `§¶½!@£¤$%€&¥/=?+\^~*´_:,<>|«»©“”µªßðđŋħĸłøæåöäþœ→↓←þ®€ł@`

func NewLexer(reader io.Reader) *schemeLexer {
	s := bufio.NewScanner(reader)
	lexer := &schemeLexer{scanner: s, pos: &Position{1, 1}}
	s.Split(makePositionAwareSplitter(lexer.pos))
	return lexer
}

func NewDebugLexer(reader io.Reader) *schemeLexer {
	lex := NewLexer(reader)
	lex.Debug = true
	return lex
}

// countWhiteSpaces counts whitespace bytes without modifying position.
// Returns (byteCount, newlineCount, colAfterLastNewline).
func countWhiteSpaces(data []byte) (munched int, newlines int, colOffset int) {
	colOffset = 0
	for ; munched < len(data); munched++ {
		if data[munched] == '\n' {
			newlines++
			colOffset = 0
		} else if unicode.IsSpace(rune(data[munched])) {
			colOffset++
		} else {
			return munched, newlines, colOffset
		}
	}
	return munched, newlines, colOffset
}

// applyWhitespaceToPos updates pos based on counted whitespace.
func applyWhitespaceToPos(pos *Position, newlines int, colOffset int) {
	if newlines > 0 {
		pos.Row += newlines
		pos.Col = 1 + colOffset
	} else {
		pos.Col += colOffset
	}
}

func isContainerChar(chr byte) bool {
	return chr == '\'' || chr == '(' || chr == ')'
}

func isStringStart(chr byte) bool {
	return chr == '`' || chr == '"'
}

func isSpace(chr byte) bool {
	return unicode.IsSpace(rune(chr))
}

func readToNewLine(data []byte, munched int, atEOF bool) (int, []byte, error) {
	for i := 1 + munched; i < len(data); i++ {
		if data[i] == '\n' {
			return i, data[munched:i], nil
		}
	}

	// If at EOF, return what we have; otherwise request more data
	if atEOF {
		return len(data), data[munched:], nil
	}
	return 0, nil, nil
}

func readString(data []byte, munched int) (int, []byte, error) {
	for i := 1 + munched; i < len(data); i++ {
		if data[i] == '"' && data[i-1] != '\\' {
			return i + 1, data[munched : i+1], nil
		}
		if data[i] == '\n' {

			return 0, data, ErrNewLineInString
		}
	}
	return 0, data, ErrEOFInString
}

func readRawString(data []byte, munched int) (int, []byte, error) {
	for i := 1 + munched; i < len(data); i++ {
		if data[i] == '`' {
			return i + 1, data[munched : i+1], nil
		}
	}
	return 0, data, ErrEOFInString
}

func readValue(data []byte, munched int, atEOF bool) (int, []byte, error) {
	for i := munched; i < len(data); i++ {
		chr := data[i]
		if isSpace(chr) || isStringStart(chr) || isContainerChar(chr) {
			return i, data[munched:i], nil
		}
	}

	// If we're at EOF, return whatever we have
	if atEOF {
		return len(data), data[munched:], nil
	}
	// Otherwise, request more data - the token might continue
	return 0, nil, nil
}

func makePositionAwareSplitter(pos *Position) func([]byte, bool) (int, []byte, error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if len(data) == 0 {
			if atEOF {
				return 0, nil, bufio.ErrFinalToken
			}
			// Request more data
			return 0, nil, nil
		}

		// Count whitespace without modifying position yet
		var munched, newlines, colOffset int
		if isSpace(data[0]) {
			munched, newlines, colOffset = countWhiteSpaces(data)
		}

		if munched < len(data) {
			first := data[munched]

			if first == ';' {
				adv, tok, err := readToNewLine(data, munched, atEOF)
				if tok != nil || err != nil {
					// Successfully got a token or final token - apply position changes
					applyWhitespaceToPos(pos, newlines, colOffset)
				}
				return adv, tok, err
			}

			if first == '"' {
				adv, tok, err := readString(data, munched)
				if tok != nil || err != nil {
					applyWhitespaceToPos(pos, newlines, colOffset)
				}
				return adv, tok, err
			}
			if first == '`' {
				adv, tok, err := readRawString(data, munched)
				if tok != nil || err != nil {
					applyWhitespaceToPos(pos, newlines, colOffset)
				}
				return adv, tok, err
			}
			if isContainerChar(first) {
				applyWhitespaceToPos(pos, newlines, colOffset)
				return 1 + munched, data[munched : munched+1], nil
			}
			if first == '#' {
				if munched+1 >= len(data) {
					// Need more data to determine what follows #
					if atEOF {
						// Lone # at EOF - treat as unexpected token
						applyWhitespaceToPos(pos, newlines, colOffset)
						return len(data), data[munched:], nil
					}
					// Don't apply position changes - we're requesting more data
					return 0, nil, nil
				}
				second := data[munched+1]
				if second == '!' {
					adv, tok, err := readToNewLine(data, munched, atEOF)
					if tok != nil || err != nil {
						applyWhitespaceToPos(pos, newlines, colOffset)
					}
					return adv, tok, err
				}
				if second == 't' || second == 'f' {
					applyWhitespaceToPos(pos, newlines, colOffset)
					return 2 + munched, data[munched : munched+2], nil
				}
			}
			adv, tok, err := readValue(data, munched, atEOF)
			if tok != nil || err != nil {
				applyWhitespaceToPos(pos, newlines, colOffset)
			}
			return adv, tok, err
		}

		// All data was whitespace
		if atEOF {
			// Apply position changes since we're done
			applyWhitespaceToPos(pos, newlines, colOffset)
			return munched, nil, bufio.ErrFinalToken
		}
		// Request more data, advancing past whitespace we've counted
		// Apply position changes since we're advancing
		applyWhitespaceToPos(pos, newlines, colOffset)
		return munched, nil, nil
	}
}
