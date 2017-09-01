package macromancy

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/parser"
)

func TestMacroGroupMatchingIdentifier(t *testing.T) {

	cases := []struct {
		in         string
		identifier string
		macros     []Macro
	}{
		{"foo", "foo", []Macro{Macro{}}},
		{"bar", "foo", nil},
		{"(() foo)", "foo", nil},
		{"bar", "bar", []Macro{Macro{}}},
		{"(bar)", "bar", []Macro{Macro{}}},
		{"(bar a b c)", "bar", []Macro{Macro{}}},
	}
	for _, c := range cases {

		codeOk, code := parser.Parse(strings.NewReader(c.in))
		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		group := MacroGroup{e.Identifier(c.identifier), c.macros}

		if len(group.Matches(code.Expressions.Head())) != len(c.macros) {
			t.Errorf("Expected match of '%s' with '%s' to yield %v", c.in, c.identifier, c.macros)
		}
	}
}

func TestBuildMacroGroupFromCode(t *testing.T) {
	cases := []struct {
		in             string
		identifier     string
		expectedMacros []Macro
	}{
		{`(define-syntax an-id (syntax-rules () () ))`, "an-id", nil},
		{`(define-syntax foo (syntax-rules () () ))`, "foo", nil},
		{`(define-syntax foo (syntax-rules () ((foo bar)) ))`, "foo",
			[]Macro{
				Macro{e.Identifier("foo"), e.Identifier("bar")},
			},
		},
		{`(define-syntax foo (syntax-rules () (((foo fiz 1) (fiz foo)) (foo bar)) ))`, "foo",
			[]Macro{
				Macro{
					e.Cons(e.Identifier("foo"), e.Cons(e.Identifier("fiz"), e.Cons(e.Integer(1), e.NIL))),
					e.Cons(e.Identifier("fiz"), e.Cons(e.Identifier("foo"), e.NIL)),
				},
				Macro{e.Identifier("foo"), e.Identifier("bar")},
			},
		},
	}
	for _, c := range cases {
		codeOk, code := parser.Parse(strings.NewReader(c.in))
		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		actual, err := NewMacroGroup(code.Expressions.Head())

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

func TestFailingCodeForBuildingMacroGroups(t *testing.T) {
	cases := []struct {
		in     string
		errMsg string
	}{
		{`(define-syntax (an-id) (syntax-rules () () ))`, "Identifier for macro group '(an-id)' is invalid."},
		{`(define-syntax 1.2 (syntax-rules () () ))`, "Identifier for macro group '1.2' is invalid."},
		{`(define-syntax . 1.2)`, "Invalid syntax definition."},
		{`define-syntax`, "Invalid syntax definition."},
		{`(define-syntax a)`, "Invalid syntax-rules."},
		{`(define-syntax a . (syntax-rules))`, "Invalid syntax-rules."},
		{`(define-syntax a (syntax-rules))`, "Invalid syntax-rules."},
		{`(define-syntax a (rules-syntax))`, "Invalid syntax-rules."},
		{`(define-syntax a . syntax-rules)`, "Invalid syntax-rules."},
		{`(define-syntax a (syntax-rules ()))`, "Invalid rules in syntax definition."},
		{`(define-syntax a (syntax-rules foo))`, "Invalid rules in syntax definition."},
		{`(define-syntax a (syntax-rules . foo))`, "Invalid rules in syntax definition."},
		{`(define-syntax a (syntax-rules () . foo))`, "Invalid rules in syntax definition."},
		{`(define-syntax a (syntax-rules () (foo bar)))`, "Invalid rule definition."},
		{`(define-syntax a (syntax-rules () ((foo))))`, "Invalid rule definition."},
		{`(define-syntax a (syntax-rules () ((foo . bar))))`, "Invalid rule definition."},
		{`(define-syntax a (syntax-rules () ((foo bar) (foo . bar))))`, "Invalid rule definition."},
		{`(define-syntax a (syntax-rules () ((foo bar) foo)))`, "Invalid rule definition."},
		{`(define-syntax a (syntax-rules () ((foo bar) . foo)))`, "Invalid rule definition."},
	}
	for _, c := range cases {
		codeOk, code := parser.Parse(strings.NewReader(c.in))
		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		_, err := NewMacroGroup(code.Expressions.Head())

		if err == nil {
			t.Errorf("Expected error '%s', but got nil when building macro group from '%s'", c.errMsg, c.in)
		} else if err.Error() != c.errMsg {
			t.Errorf("Got unexpected error message '%s'. Expected '%s' when building macro group from '%s'", err, c.errMsg, c.in)
		}
	}
}
