package macromancy

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/parser"
)

func TestMacrosCanMatchAnExpression(t *testing.T) {
	cases := []struct {
		in      string
		pattern e.Expr
	}{
		{"foo", e.Identifier("foo")},
		{"(bar)", e.Pair{e.Identifier("bar"), e.NIL}},
		{"(baz 1)", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.NIL}}},
	}

	for _, c := range cases {

		macro := Macro{c.pattern}

		ok, parseRes := parser.Parse(strings.NewReader(c.in))

		if ok != 0 {
			t.Fatal("Parsing code failed")
		}

		if !macro.Matches(parseRes.Expressions.Head()) {
			t.Errorf(`Macro %s did not match %s`, c.pattern.Repr(), c.in)
		}

	}
}

func TestMacrosDoesNotMatchNonMatchingPatterns(t *testing.T) {
	cases := []struct {
		in      string
		pattern e.Expr
	}{
		{"(foo)", e.Identifier("foo")},
		{"bar", e.Pair{e.Identifier("bar"), e.NIL}},
		{"(baz 1 x)", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.NIL}}},
	}

	for _, c := range cases {

		macro := Macro{c.pattern}

		ok, parseRes := parser.Parse(strings.NewReader(c.in))

		if ok != 0 {
			t.Fatal("Parsing code failed")
		}

		if macro.Matches(parseRes.Expressions.Head()) {
			t.Errorf(`Macro %s matched code "%s" which it shouldn't`, c.pattern.Repr(), c.in)
		}

	}
}
