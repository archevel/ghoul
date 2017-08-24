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
		pattern string
	}{
		{"foo", "foo"},
		{"(bar)", "(bar)"},
		{"(baz 1)", "(baz x)"},
		{"(numbers 1 2 3)", "(numbers x y z)"},
		{"(numbers 1 2 3)", "(numbers . x)"},
		{"(numbers 1 2 3)", "(numbers x . y)"},
		{"(numbers 1 2 3)", "(numbers x y . z)"},
		{"(numbers 1 2 3)", "(numbers x y z . å)"},
		{"(numbers 1 2 3)", "(numbers ...)"},
		{"(zoom 1 (love 'foo))", "(zoom x (zoomer z))"},
	}
	for _, c := range cases {
		patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
		if patternOk != 0 {
			t.Fatalf("Parsing pattern '%s' failed", c.pattern)
		}

		macro := Macro{pattern.Expressions.Head(), nil}

		codeOk, code := parser.Parse(strings.NewReader(c.in))

		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if ok, _ := macro.Matches(code.Expressions.Head()); !ok {
			t.Errorf(`Macro %s did not match %s`, c.pattern, c.in)
		}

	}
}

func TestMacrosBindCorrectlyCommonPatterns(t *testing.T) {
	cases := []struct {
		in               string
		pattern          string
		expectedBindings bindings
	}{
		{"foo", "foo", nil},
		{"(bar)", "(bar)", nil},
		{"(baz 1)", "(baz x)", bindings{
			e.Identifier("x"): e.Integer(1),
		}},
		{"(baz 1 `foo`)", "(baz x y)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.String("foo"),
		}},
		{"(zoom 1 (love 'foo))", "(zoom x (zoomer z))", bindings{
			e.Identifier("x"):      e.Integer(1),
			e.Identifier("zoomer"): e.Identifier("love"),
			e.Identifier("z"):      e.Quote{e.Identifier("foo")},
		}},
		{"(numbers 1 2 3)", "(numbers x y z)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Integer(2),
			e.Identifier("z"): e.Integer(3),
		}},
		{"(numbers 1 2 . 3)", "(numbers x y z)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Integer(2),
			e.Identifier("z"): e.Integer(3),
		}},
		{"(numbers 1 2 3)", "(numbers . x)", bindings{
			e.Identifier("x"): e.Pair{e.Integer(1), e.Pair{e.Integer(2), e.Pair{e.Integer(3), e.NIL}}},
		}},
		{"(numbers 1 2 . 3)", "(numbers . x)", bindings{
			e.Identifier("x"): e.Pair{e.Integer(1), e.Pair{e.Integer(2), e.Integer(3)}},
		}},
		{"(numbers 1 2 3)", "(numbers x . y)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Pair{e.Integer(2), e.Pair{e.Integer(3), e.NIL}},
		}},
		{"(numbers 1 2 . 3)", "(numbers x . y)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Pair{e.Integer(2), e.Integer(3)},
		}},
		{"(numbers 1 2 3)", "(numbers x y . z)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Integer(2),
			e.Identifier("z"): e.Pair{e.Integer(3), e.NIL},
		}},
		{"(numbers 1 2 . 3)", "(numbers x y . z)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Integer(2),
			e.Identifier("z"): e.Integer(3),
		}},
		{"(numbers 1 2 3)", "(numbers x y z . å)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Integer(2),
			e.Identifier("z"): e.Integer(3),
			e.Identifier("å"): e.NIL,
		}},
		{"(define (love foo za ba) foo bar 1)", "(define (f . a_1) . a_2)", bindings{
			e.Identifier("f"):   e.Identifier("love"),
			e.Identifier("a_1"): e.Pair{e.Identifier("foo"), e.Pair{e.Identifier("za"), e.Pair{e.Identifier("ba"), e.NIL}}},
			e.Identifier("a_2"): e.Pair{e.Identifier("foo"), e.Pair{e.Identifier("bar"), e.Pair{e.Integer(1), e.NIL}}},
		}},
	}

	for _, c := range cases {

		patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
		if patternOk != 0 {
			t.Fatalf("Parsing pattern '%s' failed", c.pattern)
		}

		macro := Macro{pattern.Expressions.Head(), nil}

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))

		if parseOk != 0 {
			t.Fatal("Parsing code failed")
		}
		_, bindings := macro.Matches(parseRes.Expressions.Head())
		if len(bindings) != len(c.expectedBindings) {
			t.Errorf(`Macro %s did not bind corretly for %s. Expected %d bindings got %d`,
				c.pattern, c.in, len(c.expectedBindings), len(bindings))
		}

		for k, expectedValue := range c.expectedBindings {
			value := bindings[k]
			if value == nil {
				t.Errorf("Expected value %s for key %s in %s using %s, but got nil!", expectedValue.Repr(), k)
			} else if !expectedValue.Equiv(value) {
				t.Errorf("Expected value %s for key %s in %s using %s in bindings, got %s",
					expectedValue.Repr(), k.Repr(), c.in, c.pattern, value.Repr())
			}
		}

		for k, value := range bindings {
			if !value.Equiv(c.expectedBindings[k]) {
				t.Errorf("Found value %s for key %s in macro bindings that is not present in the expected bindings", value.Repr(), k)
			}
		}
	}
}

func TestMacrosBindCorrectlyWithElipsisPattern(t *testing.T) {
	cases := []struct {
		in               string
		pattern          string
		expectedBindings bindings
	}{
		{"(numbers 1 2 3)", "(numbers ...)", bindings{
			e.Identifier("..."): e.Pair{e.Integer(1), e.Pair{e.Integer(2), e.Pair{e.Integer(3), e.NIL}}},
		}},
		{"(numbers 1 2 3)", "(numbers x ...)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Pair{e.Integer(2), e.Pair{e.Integer(3), e.NIL}},
		}},
		{"(numbers 1 2 . 3)", "(numbers ...)", bindings{
			e.Identifier("..."): e.Pair{e.Integer(1), e.Pair{e.Integer(2), e.Integer(3)}},
		}},

		// tail bindings
		{"(numbers 1 2 3)", "(numbers x ... y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Pair{e.Integer(2), e.NIL},
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 2 3)", "(numbers x ... z y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 2 3)", "(numbers x z ... y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Pair{e.Integer(3), e.NIL},
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.NIL,
		}},

		// tail bindings with pair code
		{"(numbers 1 2 . 3)", "(numbers x ... y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Pair{e.Integer(2), e.NIL},
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 2 . 3)", "(numbers x ... z y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 2 . 3)", "(numbers x z ... y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Integer(3),
		}},

		// final is dot patterns
		{"(numbers 1 2 3)", "(numbers x ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Pair{e.Integer(2), e.NIL},
			e.Identifier("y"):   e.Pair{e.Integer(3), e.NIL},
		}},
		{"(numbers 1 2 3)", "(numbers x ... z . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,

			e.Identifier("z"): e.Integer(2),
			e.Identifier("y"): e.Pair{e.Integer(3), e.NIL},
		}},
		{"(numbers 1 2 3)", "(numbers x z ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Pair{e.Integer(3), e.NIL},
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.NIL,
		}},

		// final is dot patterns and code has dot too
		{"(numbers 1 2 . 3)", "(numbers x ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Pair{e.Integer(2), e.NIL},
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 2 . 3)", "(numbers x ... z . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 2 . 3)", "(numbers x z ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Integer(3),
		}}, /**/

	}

	for _, c := range cases {

		patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
		if patternOk != 0 {
			t.Fatalf("Parsing pattern '%s' failed", c.pattern)
		}

		macro := Macro{pattern.Expressions.Head(), nil}

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))

		if parseOk != 0 {
			t.Fatalf("Parsing code %s failed", c.in)
		}
		_, bindings := macro.Matches(parseRes.Expressions.Head())
		if len(bindings) != len(c.expectedBindings) {
			t.Errorf(`Macro %s did not bind corretly for %s. Expected %d bindings got %d`,
				c.pattern, c.in, len(c.expectedBindings), len(bindings))
		}

		for k, expectedValue := range c.expectedBindings {
			value := bindings[k]
			if value == nil {
				t.Errorf("Expected value %s for key %s in %s using %s, but got nil!", expectedValue.Repr(), k)
			} else if !expectedValue.Equiv(value) {
				t.Errorf("Expected value %s for key %s in %s using %s in bindings, got %s",
					expectedValue.Repr(), k.Repr(), c.in, c.pattern, value.Repr())
			}
		}

		for k, value := range bindings {
			if !value.Equiv(c.expectedBindings[k]) {
				t.Errorf("Found value %s for key %s in macro bindings that is not present in the expected bindings", value.Repr(), k)
			}
		}
	}
}

func TestMacrosDoesNotMatchNonMatchingPatterns(t *testing.T) {
	cases := []struct {
		in      string
		pattern string
	}{
		{"(foo)", "foo"},
		{"bar", "(bar)"},
		{"(baz 1 x)", "(baz x)"},
		{"(baz)", "(baz x)"},
		{"(zoom 1 (love 'foo))", "(zoom x (zoomer))"},
		{"(numbers 1 2 . 3)", "(numbers x y z . å)"},
		{"(define (love foo za ba) foo bar 1)", "(define (f . a_1) a_2)"},
	}

	for _, c := range cases {
		patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
		if patternOk != 0 {
			t.Fatal("Parsing pattern failed")
		}

		macro := Macro{pattern.Expressions.Head(), nil}

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))
		if parseOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if ok, _ := macro.Matches(parseRes.Expressions.Head()); ok {
			t.Errorf(`Macro %s matched code "%s" which it shouldn't`, c.pattern, c.in)
		}

	}
}

func TestMacroExpansion(t *testing.T) {
	cases := []struct {
		expectedRepr string
		body         string
		bound        bindings
	}{
		{"foo", "foo", nil},
		{"(bar)", "(bar)", nil},

		{"(baz 1)", "(baz x)", bindings{
			e.Identifier("x"): e.Integer(1),
		}},
		{"(baz 1 \"foo\")", "(baz x y)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.String("foo"),
		}},
	}

	for _, c := range cases {

		bodyOk, body := parser.Parse(strings.NewReader(c.body))
		if bodyOk != 0 {
			t.Fatal("Parsing pattern failed")
		}

		macro := Macro{nil, body.Expressions.Head()}

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

	_, pattern := parser.Parse(strings.NewReader("(swap x y)"))
	_, body := parser.Parse(strings.NewReader("(let ((tmp x)) (set! x y) (set! y tmp))"))
	_, code := parser.Parse(strings.NewReader("(swap foo bar)"))

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
		in      string
		pattern string
		body    string
		out     string
	}{
		{
			`(define (foo x) x)`, `(define (f . params) . bdy)`, `(define f (lambda params . bdy))`, `(define foo (lambda (x) x))`,
		},
	}
	for _, c := range cases {
		patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
		if patternOk != 0 {
			t.Fatal("Parsing pattern failed")
		}

		bodyOk, body := parser.Parse(strings.NewReader(c.body))
		if bodyOk != 0 {
			t.Fatal("Parsing pattern failed")
		}

		macro := Macro{pattern.Expressions.Head(), body.Expressions.Head()}

		codeOk, code := parser.Parse(strings.NewReader(c.in))

		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		bindOk, bound := macro.Matches(code.Expressions.(e.List).Head())

		if !bindOk {
			t.Errorf("Could not bind %s to patterns in %s", c.in, c.pattern)
		}

		res, err := macro.Expand(bound)

		if err != nil {
			t.Errorf("Could not expand %s into %s got error %s", c.in, c.out, err)
		}

		if res.Repr() != c.out {
			t.Errorf("Expansion of %s did not give expected result %s, instead got %+v", c.in, c.out, res.Repr())
		}
	}

}
