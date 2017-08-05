package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	sc "text/scanner"
	"unicode"

	e "github.com/archevel/ghoul/expressions"
)

var ErrNewLineInString = errors.New("New line in string")
var ErrEOFInString = errors.New("eof in string")

type Position struct {
	Row int
	Col int
}

func (p Position) String() string {
	return fmt.Sprintf("(%d:%d)", p.Row, p.Col)
}

type schemeLexer struct {
	scanner          *bufio.Scanner
	lpair            e.List
	pos              *Position
	PairSrcPositions map[e.Pair]Position
}

func (l schemeLexer) SetPairSrcPosition(pair *e.Pair, pos Position) {
	l.PairSrcPositions[*pair] = pos
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

func (l schemeLexer) Lex(lval *yySymType) int {

	scanToNextNonComment(l.scanner)

	if l.scanner.Err() != nil {
		//TODO: Handle error...
		lval.tok = l.scanner.Text()
		lval.col = l.pos.Col
		lval.row = l.pos.Row
		return UNEXPECTED_TOKEN
	}

	data := l.scanner.Bytes()
	lval.col = l.pos.Col
	lval.row = l.pos.Row

	l.pos.Col += len(data)
	if len(data) > 0 {
		first := data[0]
		switch first {
		case '`':
			lastNewLine := bytes.LastIndexByte(data, '\n')

			if lastNewLine >= 0 {
				l.pos.Col = (len(data) - lastNewLine)
			}

			l.pos.Row += bytes.Count(data, []byte{'\n'})
			fallthrough
		case '"':
			tok, _ := strconv.Unquote(l.scanner.Text())
			lval.tok = tok
			return STRING
		case '(':
			return BEG_LIST
		case ')':
			return END_LIST
		case '\'':
			return QUOTE
		case '.':
			return DOT
		case '#':
			if len(data) > 1 {
				second := data[1]
				if second == 't' {
					return TRUE
				} else if second == 'f' {
					return FALSE
				} else if second == '!' {
					return HASHBANG
				}
			} else {
				lval.tok = l.scanner.Text()
				return UNEXPECTED_TOKEN
			}
		default:
			return handleValue(lval, data)

		}
		lval.tok = l.scanner.Text()
		return UNEXPECTED_TOKEN
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
	return UNEXPECTED_TOKEN, buffer.String()
}

func (l schemeLexer) Error(e string) {
	fmt.Printf("Error: %s", e)
}

const SPECIAL_IDENTIFIERS = `§¶½!@£¤$%€&¥/=?+\^~*´_:,<>|«»©“”µªßðđŋħĸłøæåöäþœ→↓←þ®€ł@`

func NewLexer(reader io.Reader) *schemeLexer {
	s := bufio.NewScanner(reader)
	pairSrcPositions := make(map[e.Pair]Position)
	lexer := schemeLexer{s, e.NIL, &Position{1, 1}, pairSrcPositions}
	s.Split(makePositionAwareSplitter(lexer.pos))

	return &lexer
}

func eatWhiteSpaces(data []byte, pos *Position) int {

	munched := 0
	for ; munched < len(data); munched++ {
		if data[munched] == '\n' {
			pos.Row++
			pos.Col = 1
		} else if unicode.IsSpace(rune(data[munched])) {
			pos.Col++
		} else {
			return munched
		}
	}

	return munched
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

func readToNewLine(data []byte, munched int) (int, []byte, error) {
	for i := 1 + munched; i < len(data); i++ {
		if data[i] == '\n' {
			return i, data[munched:i], nil
		}
	}

	return len(data), data[munched:], nil
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

func readValue(data []byte, munched int) (int, []byte, error) {
	for i := munched; i < len(data); i++ {
		chr := data[i]
		if isSpace(chr) || isStringStart(chr) || isContainerChar(chr) {
			return i, data[munched:i], nil
		}
	}

	return len(data), data[munched:], nil
}

func makePositionAwareSplitter(pos *Position) func([]byte, bool) (int, []byte, error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if len(data) == 0 {
			return 0, data[:0], bufio.ErrFinalToken
		}
		var munched = 0
		if isSpace(data[0]) {
			munched = eatWhiteSpaces(data, pos)
		}

		if munched < len(data) {
			first := data[munched]

			if first == ';' {
				return readToNewLine(data, munched)
			}

			if first == '"' {
				return readString(data, munched)
			}
			if first == '`' {
				return readRawString(data, munched)
			}
			if isContainerChar(first) {
				return 1 + munched, data[munched : munched+1], nil
			}
			if first == '#' && munched+1 < len(data) {
				second := data[munched+1]
				if second == '!' {
					readToNewLine(data, munched)
				}
				if second == 't' || second == 'f' {
					return 2 + munched, data[munched : munched+2], nil
				}
			}

			return readValue(data, munched)
		}

		return 0, data[:0], bufio.ErrFinalToken
	}
}
