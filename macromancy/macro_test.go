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

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))

		if parseOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if ok, _ := macro.Matches(parseRes.Expressions.Head()); !ok {
			t.Errorf(`Macro %s did not match %s`, c.pattern.Repr(), c.in)
		}

	}
}

func TestMacrosBindCorrectly(t *testing.T) {
	cases := []struct {
		in               string
		pattern          e.Expr
		expectedBindings map[e.Identifier]e.Expr
	}{
		{"foo", e.Identifier("foo"), nil},
		{"(bar)", e.Pair{e.Identifier("bar"), e.NIL}, nil},

		{"(baz 1)", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.NIL}}, map[e.Identifier]e.Expr{
			e.Identifier("x"): e.Integer(1),
		}},
		{"(baz 1 `foo`)", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.Pair{e.Identifier("y"), e.NIL}}}, map[e.Identifier]e.Expr{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.String("foo"),
		}},
	}

	for _, c := range cases {

		macro := Macro{c.pattern}

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))

		if parseOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if _, bindings := macro.Matches(parseRes.Expressions.Head()); len(bindings) != len(c.expectedBindings) {
			t.Errorf(`Macro %s did not bind corretly for %s. Expected %d bindings got %d`,
				c.pattern.Repr(), c.in, len(c.expectedBindings), len(bindings))
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
		{"(baz)", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.NIL}}},
	}

	for _, c := range cases {

		macro := Macro{c.pattern}

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))

		if parseOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if ok, _ := macro.Matches(parseRes.Expressions.Head()); ok {
			t.Errorf(`Macro %s matched code "%s" which it shouldn't`, c.pattern.Repr(), c.in)
		}

	}
}

type DefineMacro struct{}

func (m DefineMacro) Transform(expr e.Expr) (e.Expr, error) {
	list := expr.(e.List)
	define := list.Head()
	list = list.Tail().(e.List)
	idAndArgs := list.Head().(e.List)
	id := idAndArgs.Head()
	args := idAndArgs.Tail()
	body := list.Tail()

	return e.Pair{define, e.Pair{id, e.Pair{e.Pair{e.Identifier("lambda"), e.Pair{args, body}}, e.NIL}}}, nil
}

func TestMacroTransform(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{
			`(define (foo x) x)`, `(define foo (lambda (x) x))`,
		},
	}
	for _, c := range cases {

		macro := DefineMacro{}

		ok, parseRes := parser.Parse(strings.NewReader(c.in))

		if ok != 0 {
			t.Fatal("Parsing code failed")
		}

		res, err := macro.Transform(parseRes.Expressions.Head())

		if err != nil {
			t.Errorf("Could not transform %s into %s\n", c.in, c.out)
		}

		if res.Repr() != c.out {
			t.Errorf("Transform of %s did not give expected result %s, instead got %+v", c.in, c.out, res.Repr())
		}
	}

}
