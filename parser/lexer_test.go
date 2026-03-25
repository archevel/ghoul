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

func TestLexerHandlesUTF8InComments(t *testing.T) {
	// Comments containing multi-byte UTF-8 characters (em-dash, etc.)
	// must be skipped without causing parse errors.
	cases := []struct {
		name string
		in   string
		want []int
	}{
		{"em-dash in comment", ";; let \u2014 local bindings\n42", []int{INTEGER}},
		{"indented comment with em-dash", "  ;; foo \u2014 bar\n42", []int{INTEGER}},
		{"CJK in comment", ";; \u4f60\u597d\n42", []int{INTEGER}},
		{"emoji in comment", ";; test \U0001F600\n42", []int{INTEGER}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := strings.NewReader(c.in)
			var lexer yyLexer = NewLexer(r)
			for i, expected := range c.want {
				lval := yySymType{}
				actual := lexer.Lex(&lval)
				if actual != expected {
					t.Errorf("Expected token %d at position %d, got %d (text: %q)", expected, i, actual, lval.tok)
				}
			}
		})
	}
}

func TestLexerHandlesUTF8InIdentifiers(t *testing.T) {
	// Identifiers can contain Unicode letters (already supported via
	// SPECIAL_IDENTIFIERS and unicode.IsLetter in isIdentRune).
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"Swedish chars", "\u00e5\u00e4\u00f6", "\u00e5\u00e4\u00f6"},
		{"section sign", "\u00a7foo", "\u00a7foo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := strings.NewReader(c.in)
			var lexer yyLexer = NewLexer(r)
			lval := yySymType{}
			tok := lexer.Lex(&lval)
			if tok != IDENTIFIER {
				t.Errorf("Expected IDENTIFIER, got %d (text: %q)", tok, lval.tok)
			}
			if lval.tok != c.want {
				t.Errorf("Expected identifier %q, got %q", c.want, lval.tok)
			}
		})
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
