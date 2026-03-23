package macromancy

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/parser"
)

func TestBuildMacroGroupFromCode(t *testing.T) {
	cases := []struct {
		in             string
		identifier     string
		expectedMacros []Macro
	}{
		{`(define-syntax an-id (syntax-rules () () ))`, "an-id", nil},
		{`(define-syntax foo (syntax-rules () () ))`, "foo", nil},
		{`(define-syntax foo (syntax-rules () (foo bar) ))`, "foo",
			[]Macro{
				{Pattern: e.Identifier("foo"), Body: e.Identifier("bar")},
			},
		},
		{`(define-syntax foo (syntax-rules () ((foo fiz 1) (fiz foo)) (foo bar) ))`, "foo",
			[]Macro{
				{
					Pattern: e.Cons(e.Identifier("foo"), e.Cons(e.Identifier("fiz"), e.Cons(e.Integer(1), e.NIL))),
					Body:    e.Cons(e.Identifier("fiz"), e.Cons(e.Identifier("foo"), e.NIL)),
				},
				{Pattern: e.Identifier("foo"), Body: e.Identifier("bar")},
			},
		},
	}
	for _, c := range cases {
		codeOk, code := parser.Parse(strings.NewReader(c.in))
		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		actual, err := NewMacroGroup(code.Expressions.First())

		if err != nil {
			t.Errorf("Got unexpected error '%s' when creating new macro group from '%s'", err, c.in)
		}

		if !actual.matchId.Equiv(e.Identifier(c.identifier)) {
			t.Errorf("Expected matchId to be '%s', but was '%s', after building macro group from %s", c.identifier, actual.matchId, c.in)
		}

		if actual.macros == nil {
			t.Errorf("Got nil as macros, expected to build non-nil list of macros from %s", c.in)
		} else if len(actual.macros) != len(c.expectedMacros) {
			t.Errorf("Not expected number of Macros from '%s'", c.in)
		} else if c.expectedMacros != nil {
			for i, m := range c.expectedMacros {
				if !m.Pattern.Equiv(actual.macros[i].Pattern) {
					t.Errorf("Got pattern '%s' expected '%s'", actual.macros[i].Pattern.Repr(), m.Pattern.Repr())
				}
				if !m.Body.Equiv(actual.macros[i].Body) {
					t.Errorf("Got pattern '%s' expected '%s'", actual.macros[i].Body.Repr(), m.Body.Repr())
				}
			}
		}
	}
}

func TestMacroGroupMacrosAccessor(t *testing.T) {
	code := `(define-syntax foo (syntax-rules () (foo bar)))`
	codeOk, parsed := parser.Parse(strings.NewReader(code))
	if codeOk != 0 {
		t.Fatal("Parsing code failed")
	}
	mg, err := NewMacroGroup(parsed.Expressions.First())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	macros := mg.Macros()
	if len(macros) != 1 {
		t.Errorf("expected 1 macro, got %d", len(macros))
	}
}

func TestBuildMacroGroupWithLiterals(t *testing.T) {
	code := `(define-syntax foo (syntax-rules (bar baz) (foo 1)))`
	codeOk, parsed := parser.Parse(strings.NewReader(code))
	if codeOk != 0 {
		t.Fatal("Parsing code failed")
	}
	mg, err := NewMacroGroup(parsed.Expressions.First())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	macros := mg.Macros()
	if len(macros) != 1 {
		t.Fatalf("expected 1 macro, got %d", len(macros))
	}
	if !macros[0].Literals[e.Identifier("bar")] || !macros[0].Literals[e.Identifier("baz")] {
		t.Error("expected literals 'bar' and 'baz' to be set")
	}
}

func TestBuildMacroGroupMultipleRules(t *testing.T) {
	code := `(define-syntax foo (syntax-rules () ((foo x y) (+ x y)) ((foo x) x)))`
	codeOk, parsed := parser.Parse(strings.NewReader(code))
	if codeOk != 0 {
		t.Fatal("Parsing code failed")
	}
	mg, err := NewMacroGroup(parsed.Expressions.First())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	macros := mg.Macros()
	if len(macros) != 2 {
		t.Errorf("expected 2 macros, got %d", len(macros))
	}
}

func TestBuildMacroGroupWithNonIdentifierLiteralFails(t *testing.T) {
	code := `(define-syntax foo (syntax-rules (42) (foo bar)))`
	codeOk, parsed := parser.Parse(strings.NewReader(code))
	if codeOk != 0 {
		t.Fatal("Parsing code failed")
	}
	_, err := NewMacroGroup(parsed.Expressions.First())
	if err == nil {
		t.Error("expected error for non-identifier in literals list")
	}
	if err != nil && !strings.Contains(err.Error(), "expected identifier") {
		t.Errorf("expected error about identifier, got: %v", err)
	}
}

func TestFailingCodeForBuildingMacroGroups(t *testing.T) {
	cases := []struct {
		in     string
		errMsg string
	}{
		{`(define-syntax (an-id) (syntax-rules () () ))`, "invalid identifier (an-id) for macro group: must be an identifier"},
		{`(define-syntax 1.2 (syntax-rules () () ))`, "invalid identifier 1.2 for macro group: must be an identifier"},
		{`(define-syntax . 1.2)`, "invalid syntax definition: expected list with syntax transformer, got (define-syntax . 1.2)"},
		{`define-syntax`, "invalid syntax definition: expected (define-syntax <identifier> <transformer>), got define-syntax"},
		{`(define-syntax a)`, "invalid syntax-rules: malformed syntax-rules structure in ()"},
		{`(define-syntax a . (syntax-rules))`, "invalid syntax-rules: expected syntax-rules form, got syntax-rules"},
		{`(define-syntax a (syntax-rules))`, "invalid syntax-rules: malformed syntax-rules structure in (syntax-rules)"},
		{`(define-syntax a (rules-syntax))`, "invalid syntax-rules: malformed syntax-rules structure in (rules-syntax)"},
		{`(define-syntax a . syntax-rules)`, "invalid syntax-rules: expected syntax-rules form, got (a . syntax-rules)"},
		{`(define-syntax a (syntax-rules ()))`, "invalid rules in syntax definition: missing rules list in (())"},
		{`(define-syntax a (syntax-rules foo))`, "invalid rules in syntax definition: missing rules list in (foo)"},
		{`(define-syntax a (syntax-rules . foo))`, "invalid rules in syntax definition: expected literals and rules, got (syntax-rules . foo)"},
		{`(define-syntax a (syntax-rules () . foo))`, "invalid rules in syntax definition: expected rules after literals, got (() . foo)"},
		{`(define-syntax a (syntax-rules () foo))`, "invalid rule definition: expected list for rule, got expressions.Identifier at position 0"},
		{`(define-syntax a (syntax-rules () (foo)))`, "invalid rule definition: rule must have pattern and body, got (foo)"},
		{`(define-syntax a (syntax-rules () (foo . bar)))`, "invalid rule definition: rule must have pattern and body, got (foo . bar)"},
		{`(define-syntax a (syntax-rules () (foo bar) (foo . bar)))`, "invalid rule definition: rule must have pattern and body, got (foo . bar)"},
		{`(define-syntax a (syntax-rules () (foo bar) foo))`, "invalid rule definition: expected list for rule, got expressions.Identifier at position 1"},
		{`(define-syntax a (syntax-rules () (foo bar) . foo))`, "invalid rule definition: malformed rules list at position 0"},
	}
	for _, c := range cases {
		codeOk, code := parser.Parse(strings.NewReader(c.in))
		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		_, err := NewMacroGroup(code.Expressions.First())

		if err == nil {
			t.Errorf("Expected error '%s', but got nil when building macro group from '%s'", c.errMsg, c.in)
		} else if err.Error() != c.errMsg {
			t.Errorf("Got unexpected error message '%s'. Expected '%s' when building macro group from '%s'", err, c.errMsg, c.in)
		}
	}
}
