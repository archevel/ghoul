package exhumer

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// smallReader wraps a reader and limits read sizes to simulate buffer boundaries
type smallReader struct {
	r         io.Reader
	chunkSize int
}

func (sr *smallReader) Read(p []byte) (n int, err error) {
	if len(p) > sr.chunkSize {
		p = p[:sr.chunkSize]
	}
	return sr.r.Read(p)
}

func TestLexerHandlesTokenAtBufferBoundary(t *testing.T) {
	// Test that identifiers spanning buffer boundaries are read correctly
	cases := []struct {
		name      string
		input     string
		chunkSize int
		expected  []string // expected token strings
	}{
		{
			name:      "identifier split at boundary",
			input:     "(define foo-bar-baz 42)",
			chunkSize: 10, // Will split "define" or other tokens
			expected:  []string{"(", "define", "foo-bar-baz", "42", ")"},
		},
		{
			name:      "multiple identifiers with small chunks",
			input:     "hello world test",
			chunkSize: 3,
			expected:  []string{"hello", "world", "test"},
		},
		{
			name:      "whitespace and tokens mixed",
			input:     "  \n  foo   \n\n  bar  ",
			chunkSize: 5,
			expected:  []string{"foo", "bar"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sr := &smallReader{r: strings.NewReader(tc.input), chunkSize: tc.chunkSize}
			lexer := NewLexer(sr)

			var tokens []string
			var lval yySymType
			for {
				tok := lexer.Lex(&lval)
				if tok == 0 {
					break
				}
				switch tok {
				case BEG_LIST:
					tokens = append(tokens, "(")
				case END_LIST:
					tokens = append(tokens, ")")
				case INTEGER, FLOAT, IDENTIFIER, STRING:
					tokens = append(tokens, lval.tok)
				}
			}

			if len(tokens) != len(tc.expected) {
				t.Errorf("expected %d tokens, got %d: %v", len(tc.expected), len(tokens), tokens)
				return
			}
			for i, exp := range tc.expected {
				if tokens[i] != exp {
					t.Errorf("token %d: expected %q, got %q", i, exp, tokens[i])
				}
			}
		})
	}
}

func TestLexerHandlesHashAtBufferBoundary(t *testing.T) {
	// Test that # followed by t/f at buffer boundary works
	cases := []struct {
		name      string
		input     string
		chunkSize int
		expected  int // expected token type
	}{
		{
			name:      "hash-t split",
			input:     "#t",
			chunkSize: 1, // # in one chunk, t in another
			expected:  TRUE,
		},
		{
			name:      "hash-f split",
			input:     "#f",
			chunkSize: 1,
			expected:  FALSE,
		},
		{
			name:      "hash-bang split",
			input:     "#!/usr/bin/ghoul\nfoo",
			chunkSize: 2, // #! might be split
			expected:  IDENTIFIER, // After hashbang, should get "foo"
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sr := &smallReader{r: strings.NewReader(tc.input), chunkSize: tc.chunkSize}
			lexer := NewLexer(sr)

			var lval yySymType
			tok := lexer.Lex(&lval)

			// For hashbang, skip it and get the next token
			if tok == HASHBANG {
				tok = lexer.Lex(&lval)
			}

			if tok != tc.expected {
				t.Errorf("expected token type %d, got %d", tc.expected, tok)
			}
		})
	}
}

func TestLexerHandlesCommentAtBufferBoundary(t *testing.T) {
	// Test that comments spanning buffer boundaries are handled
	cases := []struct {
		name      string
		input     string
		chunkSize int
		expected  string // expected identifier after comment
	}{
		{
			name:      "comment split across boundary",
			input:     "; this is a comment\nfoo",
			chunkSize: 5,
			expected:  "foo",
		},
		{
			name:      "long comment with small chunks",
			input:     "; this is a very long comment that spans multiple buffer reads\nbar",
			chunkSize: 10,
			expected:  "bar",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sr := &smallReader{r: strings.NewReader(tc.input), chunkSize: tc.chunkSize}
			lexer := NewLexer(sr)

			var lval yySymType
			// Skip comment tokens until we get an identifier
			for {
				tok := lexer.Lex(&lval)
				if tok == 0 {
					t.Fatal("unexpected EOF")
				}
				if tok == IDENTIFIER {
					if lval.tok != tc.expected {
						t.Errorf("expected %q, got %q", tc.expected, lval.tok)
					}
					break
				}
			}
		})
	}
}

func TestLexerPositionCorrectAfterBufferRefill(t *testing.T) {
	// Test that position tracking remains correct after buffer refills
	input := "aaa\nbbb\nccc"
	sr := &smallReader{r: strings.NewReader(input), chunkSize: 2}
	lexer := NewLexer(sr)

	expectedPositions := []Position{
		{1, 1}, // aaa
		{2, 1}, // bbb
		{3, 1}, // ccc
	}

	var lval yySymType
	for i, exp := range expectedPositions {
		tok := lexer.Lex(&lval)
		if tok == 0 {
			t.Fatalf("unexpected EOF at token %d", i)
		}
		actualPos := Position{lval.row, lval.col}
		if actualPos != exp {
			t.Errorf("token %d (%q): expected position %v, got %v", i, lval.tok, exp, actualPos)
		}
	}
}

func TestLexerHandlesLargeFile(t *testing.T) {
	// Generate a large file with many tokens
	var buf bytes.Buffer
	for i := 0; i < 1000; i++ {
		buf.WriteString("(define var")
		buf.WriteString(strings.Repeat("x", 50)) // Long identifier
		buf.WriteString(" ")
		buf.WriteString(fmt.Sprintf("%d", i))
		buf.WriteString(")\n")
	}

	res, parsed := Parse(&buf)
	if res != 0 {
		t.Errorf("failed to parse large file, result: %d", res)
	}
	if parsed.Expressions == nil {
		t.Error("expected non-nil expressions")
	}
}

func TestLexerHandlesWhitespaceOnlyAtBufferEnd(t *testing.T) {
	// Test case where buffer ends with only whitespace
	cases := []struct {
		name      string
		input     string
		chunkSize int
		expected  []string
	}{
		{
			name:      "trailing whitespace in separate chunk",
			input:     "foo   ",
			chunkSize: 4, // "foo " in first chunk, "  " in second
			expected:  []string{"foo"},
		},
		{
			name:      "whitespace between tokens at boundary",
			input:     "foo    bar",
			chunkSize: 5, // "foo  " then "  bar"
			expected:  []string{"foo", "bar"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sr := &smallReader{r: strings.NewReader(tc.input), chunkSize: tc.chunkSize}
			lexer := NewLexer(sr)

			var tokens []string
			var lval yySymType
			for {
				tok := lexer.Lex(&lval)
				if tok == 0 {
					break
				}
				if tok == IDENTIFIER {
					tokens = append(tokens, lval.tok)
				}
			}

			if len(tokens) != len(tc.expected) {
				t.Errorf("expected %d tokens, got %d: %v", len(tc.expected), len(tokens), tokens)
				return
			}
			for i, exp := range tc.expected {
				if tokens[i] != exp {
					t.Errorf("token %d: expected %q, got %q", i, exp, tokens[i])
				}
			}
		})
	}
}
