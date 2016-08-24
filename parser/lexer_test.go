package parser

import (
	"strings"
	"testing"
)

func TestLexerFindsSimpleCases(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{"(", []int{BEG_LIST}},
		{")", []int{END_LIST}},
		{"))((", []int{END_LIST, END_LIST, BEG_LIST, BEG_LIST}},
		{"'", []int{QUOTE}},
		{"''''", []int{QUOTE, QUOTE, QUOTE, QUOTE}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerFindsSimpleCasesWithWhitespaces(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{" (", []int{BEG_LIST}},
		{"\t)", []int{END_LIST}},
		{"\t\n )\n\n   )  (\t\t(", []int{END_LIST, END_LIST, BEG_LIST, BEG_LIST}},
		{"\n\n\n'", []int{QUOTE}},
		{"''\n\n\n\n'    '", []int{QUOTE, QUOTE, QUOTE, QUOTE}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerIgnoresComments(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{";'''fooo\n(", []int{BEG_LIST}},
		{";((((\n )", []int{END_LIST}},
		{"\t;#``brr\n );((\n\n   )  (\t\t(", []int{END_LIST, END_LIST, BEG_LIST, BEG_LIST}},
		{"\n;)\n\n'", []int{QUOTE}},
		{"''\n;)\n\n\n'    '", []int{QUOTE, QUOTE, QUOTE, QUOTE}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerFindsStrings(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{"`raw`", []int{STRING}},
		{`"normal"`, []int{STRING}},
		{`"foo" "foo" "bar" "bar"`, []int{STRING, STRING, STRING, STRING}},
		{"`foo` `raw`", []int{STRING, STRING}},
		{"`brr\n\nfoo`", []int{STRING}},
		{"`brr\n\nfoo`" + `"biz\t"`, []int{STRING, STRING}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerFailsToLexBadStrings(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{"`raw", []int{UNEXPECTED_TOKEN}},
		{`"normal`, []int{UNEXPECTED_TOKEN}},
		{`"brr` + "\n" + `foo"`, []int{UNEXPECTED_TOKEN}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerFindsLoneDot(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{".", []int{DOT}},
		{"`foo`.`foo`", []int{STRING, DOT, STRING}},
		{`. . . .`, []int{DOT, DOT, DOT, DOT}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerFindsBooleans(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{"#f", []int{FALSE}},
		{"#t", []int{TRUE}},
		{"#t#f#t#f", []int{TRUE, FALSE, TRUE, FALSE}}, // Is this illadviced... hmmm...
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerEmptyScriptsYield0(t *testing.T) {
	cases := []string{
		"",
		"\n",
		" ",
		"\t",
	}

	for i, c := range cases {
		r := strings.NewReader(c)
		var lexer yyLexer = NewLexer(r)

		lval := yySymType{}
		actual := lexer.Lex(&lval)

		if actual != 0 {
			t.Errorf("Test #%v: Expected 0 but got %v", i, actual)
		}

	}
}

func TestLexerFindsSimpleIdentifiers(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{"define", []int{IDENTIFIER}},
		{"foo", []int{IDENTIFIER}},
		{"define foo bar", []int{IDENTIFIER, IDENTIFIER, IDENTIFIER}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerFindsIntegers(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{"123", []int{INTEGER}},
		{"-123", []int{INTEGER}},
		{"0x123", []int{INTEGER}},
		{"0x123 932 55", []int{INTEGER, INTEGER, INTEGER}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerFindsFloats(t *testing.T) {
	cases := []struct {
		in          string
		lexedTokens []int
	}{
		{"123.0", []int{FLOAT}},
		{"-123.10", []int{FLOAT}},
		{"123.8e100", []int{FLOAT}},
		{"123.1e09 932.3 55.3", []int{FLOAT, FLOAT, FLOAT}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		for i, expected := range c.lexedTokens {

			lval := yySymType{}
			actual := lexer.Lex(&lval)
			if actual != expected {
				t.Errorf("Lexing %s. Expected '%v' as token nr. %d, got %v", c.in, expected, i, actual)
			}
		}

	}
}

func TestLexerRecordsPositionOfTokens(t *testing.T) {
	cases := []struct {
		in  string
		row int
		col int
	}{
		{"123.0", 1, 1},
		{"\n123.0", 2, 1},
		{" 123.0", 1, 2},
		{"\n 123.0", 2, 2},
		{" \n123.0", 2, 1},
		{" \n\n\n    123.0", 4, 5},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		lval := yySymType{}
		lexer.Lex(&lval)

		if lval.row != c.row {
			t.Errorf("Expected token to be on row %d but got row %d", c.row, lval.row)
		}

		if lval.col != c.col {
			t.Errorf("Expected token to be on col %d but got col %d", c.col, lval.col)
		}
	}
}

func TestLexerRecordsPositionOfLaterTokens(t *testing.T) {
	cases := []struct {
		in  string
		row int
		col int
	}{
		{"123.0 99", 1, 7},
		{"\n123.0\n99", 3, 1},
		{" 123.0 99", 1, 8},
		{"\n 123.0 99", 2, 8},
		{" \n123.0   \n99", 3, 1},
		{" \n\n\n    123.0 \n\n\n    99", 7, 5},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		lval := yySymType{}
		lexer.Lex(&lval)
		// Get second token
		lexer.Lex(&lval)

		if lval.row != c.row {
			t.Errorf("Expected token to be on row %d but got row %d", c.row, lval.row)
		}

		if lval.col != c.col {
			t.Errorf("Expected token to be on col %d but got col %d", c.col, lval.col)
		}
	}
}

func TestLexerRecordsPositionOfLaterTokensAfterRawStrings(t *testing.T) {
	cases := []struct {
		in  string
		row int
		col int
	}{
		{"`abc` 99", 1, 7},
		{"`abc\nfoo` 99", 2, 6},
		{"`\nabc\nfoo\n` 99", 4, 3},
		{"`\na` 99", 2, 4},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		var lexer yyLexer = NewLexer(r)
		lval := yySymType{}
		lexer.Lex(&lval)
		// Get second token
		lexer.Lex(&lval)

		if lval.row != c.row {
			t.Errorf("Expected token to be on row %d but got row %d", c.row, lval.row)
		}

		if lval.col != c.col {
			t.Errorf("Expected token to be on col %d but got col %d", c.col, lval.col)
		}
	}
}
