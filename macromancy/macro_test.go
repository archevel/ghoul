package macromancy

import (
	"fmt"
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

		macro := Macro{c.pattern, nil}

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
		expectedBindings bindings
	}{
		{"foo", e.Identifier("foo"), nil},
		{"(bar)", e.Pair{e.Identifier("bar"), e.NIL}, nil},

		{"(baz 1)", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.NIL}}, bindings{
			e.Identifier("x"): e.Integer(1),
		}},
		{"(baz 1 `foo`)", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.Pair{e.Identifier("y"), e.NIL}}}, bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.String("foo"),
		}},
	}

	for _, c := range cases {

		macro := Macro{c.pattern, nil}

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))

		if parseOk != 0 {
			t.Fatal("Parsing code failed")
		}
		_, bindings := macro.Matches(parseRes.Expressions.Head())
		if len(bindings) != len(c.expectedBindings) {
			t.Errorf(`Macro %s did not bind corretly for %s. Expected %d bindings got %d`,
				c.pattern.Repr(), c.in, len(c.expectedBindings), len(bindings))
		}

		for k, expectedValue := range c.expectedBindings {
			value := bindings[k]
			if value == nil {
				t.Errorf("Expected value %s for key %s got nil!", expectedValue.Repr(), k)
			} else if !expectedValue.Equiv(value) {
				t.Errorf("Expected value %s for key %s in bindings, got %s", expectedValue.Repr(), k.Repr(), value.Repr())
			}
		}

		for k, value := range bindings {
			if !value.Equiv(c.expectedBindings[k]) {
				t.Errorf("Found value %s for key %s in macro bindings that is not present in the expected bindings", value, k)
			}
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

		macro := Macro{c.pattern, nil}

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))

		if parseOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if ok, _ := macro.Matches(parseRes.Expressions.Head()); ok {
			t.Errorf(`Macro %s matched code "%s" which it shouldn't`, c.pattern.Repr(), c.in)
		}

	}
}

func TestMacroExpansion(t *testing.T) {
	cases := []struct {
		expectedRepr string
		body         e.Expr
		bound        bindings
	}{
		{"foo", e.Identifier("foo"), nil},
		{"(bar)", e.Pair{e.Identifier("bar"), e.NIL}, nil},

		{"(baz 1)", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.NIL}}, bindings{
			e.Identifier("x"): e.Integer(1),
		}},
		{"(baz 1 \"foo\")", e.Pair{e.Identifier("baz"), e.Pair{e.Identifier("x"), e.Pair{e.Identifier("y"), e.NIL}}}, bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.String("foo"),
		}},
	}

	for _, c := range cases {
		macro := Macro{nil, c.body}
		expanded, err := macro.Expand(c.bound)

		if err != nil {
			t.Errorf("Got error '%s' when expanding macro body %s with bindings %s", err, c.body, c.bound)
		}

		if expanded.Repr() != c.expectedRepr {
			t.Errorf("Expected %s after expanding macro, but got %s", c.expectedRepr, expanded.Repr())
		}
	}
}

func ExampleSwapMacro() {

	// (swap x y) - (let ((tmp x)) (set! x y) (set! y tmp))
	_, pattern := parser.Parse(strings.NewReader("(swap x y)"))
	_, body := parser.Parse(strings.NewReader("(let ((tmp x)) (set! x y) (set! y tmp))"))
	_, code := parser.Parse(strings.NewReader("(swap foo bar)"))

	//	fmt.Println("\npat:", pattern.Expressions.Repr(), "\nbody:", body.Expressions.Repr(), "\ncode:", code.Expressions.Repr())
	macro := Macro{pattern.Expressions.(e.List).Head(), body.Expressions.(e.List).Head()}
	_, bound := macro.Matches(code.Expressions.(e.List).Head())

	res, _ := macro.Expand(bound)
	fmt.Println(res.Repr())
	// Output:
	// (let ((tmp foo)) (set! foo bar) (set! bar tmp))
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
