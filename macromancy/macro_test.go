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

		macro := Macro{pattern.Expressions.First(), nil}

		codeOk, code := parser.Parse(strings.NewReader(c.in))

		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if ok, _ := macro.Matches(code.Expressions.First()); !ok {
			t.Errorf(`Macro %s did not match %s`, c.pattern, c.in)
		}

	}
}

func TestMacrosCanPatternMatch(t *testing.T) {
	cases := []struct {
		in      string
		pattern string
	}{
		{"(numbers 1 1)", "(numbers x x)"},
		{"(numbers 1 (a b 1))", "(numbers x (... x))"},
		{"(numbers 1.5 1.5 1.5)", "(numbers x 1.5 x)"},
		{"(numbers 1.5 'a 1.5)", "(numbers x 'a x)"},
		{"(numbers 1.5 '(a 1.5) 1.5)", "(numbers x '(a 1.5) x)"},
	}
	for _, c := range cases {
		patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
		if patternOk != 0 {
			t.Fatalf("Parsing pattern '%s' failed", c.pattern)
		}

		macro := Macro{pattern.Expressions.First(), nil}

		codeOk, code := parser.Parse(strings.NewReader(c.in))

		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if ok, _ := macro.Matches(code.Expressions.First()); !ok {
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
			e.Identifier("x"): e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Cons(e.Integer(3), e.NIL))),
		}},
		{"(numbers 1 2 . 3)", "(numbers . x)", bindings{
			e.Identifier("x"): e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Integer(3))),
		}},
		{"(numbers 1 2 3)", "(numbers x . y)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Cons(e.Integer(2), e.Cons(e.Integer(3), e.NIL)),
		}},
		{"(numbers 1 2 . 3)", "(numbers x . y)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Cons(e.Integer(2), e.Integer(3)),
		}},
		{"(numbers 1 2 3)", "(numbers x y . z)", bindings{
			e.Identifier("x"): e.Integer(1),
			e.Identifier("y"): e.Integer(2),
			e.Identifier("z"): e.Cons(e.Integer(3), e.NIL),
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
			e.Identifier("a_1"): e.Cons(e.Identifier("foo"), e.Cons(e.Identifier("za"), e.Cons(e.Identifier("ba"), e.NIL))),
			e.Identifier("a_2"): e.Cons(e.Identifier("foo"), e.Cons(e.Identifier("bar"), e.Cons(e.Integer(1), e.NIL))),
		}},
	}

	for _, c := range cases {
		runBindingTest(t, c.in, c.pattern, c.expectedBindings)
	}
}

func TestMacrosBindCorrectlyWithElipsisPattern(t *testing.T) {
	cases := []struct {
		in               string
		pattern          string
		expectedBindings bindings
	}{
		{"(numbers 1 2 3)", "(numbers ...)", bindings{
			e.Identifier("..."): e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Cons(e.Integer(3), e.NIL))),
		}},
		{"(numbers 1 2 3)", "(numbers x ...)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Cons(e.Integer(2), e.Cons(e.Integer(3), e.NIL)),
		}},
		{"(numbers 1 2 . 3)", "(numbers ...)", bindings{
			e.Identifier("..."): e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Integer(3))),
		}},

		// tail bindings
		{"(numbers 1 2 3)", "(numbers x z ... y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 2 3)", "(numbers x ... y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Cons(e.Integer(2), e.NIL),
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 2 3)", "(numbers x ... z y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Integer(3),
		}},

		// tail bindings with pair code
		{"(numbers 1 2 . 3)", "(numbers x ... y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Cons(e.Integer(2), e.NIL),
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
			e.Identifier("..."): e.Cons(e.Integer(2), e.NIL),
			e.Identifier("y"):   e.Cons(e.Integer(3), e.NIL),
		}},
		{"(numbers 1 2 3)", "(numbers x ... z . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,

			e.Identifier("z"): e.Integer(2),
			e.Identifier("y"): e.Cons(e.Integer(3), e.NIL),
		}},
		{"(numbers 1 2 3)", "(numbers x z ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.NIL,
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Cons(e.Integer(3), e.NIL),
		}},

		// final is dot patterns and code has dot too
		{"(numbers 1 2 . 3)", "(numbers x ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Cons(e.Integer(2), e.NIL),
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
		}},

		// more complex patterns
		{"(numbers 1 2 4 . 3)", "(numbers x z ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Cons(e.Integer(4), e.NIL),
			e.Identifier("z"):   e.Integer(2),
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 (2 4) 5 . 3)", "(numbers x z ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Cons(e.Integer(5), e.NIL),
			e.Identifier("z"):   e.Cons(e.Integer(2), e.Cons(e.Integer(4), e.NIL)),
			e.Identifier("y"):   e.Integer(3),
		}},
		{"(numbers 1 (2 4) 5 . 3)", "(numbers x (a  b) ... . y)", bindings{
			e.Identifier("x"):   e.Integer(1),
			e.Identifier("..."): e.Cons(e.Integer(5), e.NIL),
			e.Identifier("a"):   e.Integer(2),
			e.Identifier("b"):   e.Integer(4),
			e.Identifier("y"):   e.Integer(3),
		}},
	}

	for _, c := range cases {
		runBindingTest(t, c.in, c.pattern, c.expectedBindings)
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
		{"(numbers 1 2)", "(numbers x x)"},
	}

	for _, c := range cases {
		patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
		if patternOk != 0 {
			t.Fatal("Parsing pattern failed")
		}

		macro := Macro{pattern.Expressions.First(), nil}

		parseOk, parseRes := parser.Parse(strings.NewReader(c.in))
		if parseOk != 0 {
			t.Fatal("Parsing code failed")
		}

		if ok, _ := macro.Matches(parseRes.Expressions.First()); ok {
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

		macro := Macro{nil, body.Expressions.First()}

		expanded := macro.Expand(c.bound)

		if expanded.Repr() != c.expectedRepr {
			t.Errorf("Expected %s after expanding macro, but got %s", c.expectedRepr, expanded.Repr())
		}
	}
}

func swapMacroExample() {

	_, pattern := parser.Parse(strings.NewReader("(swap x y)"))
	_, body := parser.Parse(strings.NewReader("(let ((tmp x)) (set! x y) (set! y tmp))"))
	_, code := parser.Parse(strings.NewReader("(swap foo bar)"))

	macro := Macro{pattern.Expressions.(e.List).First(), body.Expressions.(e.List).First()}
	_, bound := macro.Matches(code.Expressions.(e.List).First())

	res := macro.Expand(bound)
	fmt.Println(res.Repr())
	// Output:
	// (let ((tmp foo)) (set! foo bar) (set! bar tmp))
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

		macro := Macro{pattern.Expressions.First(), body.Expressions.First()}

		codeOk, code := parser.Parse(strings.NewReader(c.in))

		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		bindOk, bound := macro.Matches(code.Expressions.(e.List).First())

		if !bindOk {
			t.Errorf("Could not bind %s to patterns in %s", c.in, c.pattern)
		}

		res := macro.Expand(bound)

		if res.Repr() != c.out {
			t.Errorf("Expansion of %s did not give expected result %s, instead got %+v", c.in, c.out, res.Repr())
		}
	}
}

func runBindingTest(t *testing.T, in string, patternStr string, bound bindings) {

	patternOk, pattern := parser.Parse(strings.NewReader(patternStr))
	if patternOk != 0 {
		t.Fatalf("Parsing pattern '%s' failed", pattern)
	}

	macro := Macro{pattern.Expressions.First(), nil}

	parseOk, parseRes := parser.Parse(strings.NewReader(in))

	if parseOk != 0 {
		t.Fatalf("Parsing code %s failed", in)
	}
	_, bindings := macro.Matches(parseRes.Expressions.First())
	if len(bindings) != len(bound) {
		t.Errorf(`Macro %s did not bind corretly for %s. Expected %d bindings got %d`,
			patternStr, in, len(bound), len(bindings))
	}

	for k, expectedValue := range bound {
		value := bindings[k]
		if value == nil {
			t.Errorf("Expected value %s for key %s in %s using %s, but got nil!", expectedValue.Repr(), k.Repr(), in, patternStr)
		} else if !expectedValue.Equiv(value) {
			t.Errorf("Expected value %s for key %s in %s using %s in bindings, got %s",
				expectedValue.Repr(), k.Repr(), in, patternStr, value.Repr())
		}
	}

	for k, value := range bindings {
		if !value.Equiv(bound[k]) {
			t.Errorf("Found value %s for key %s in macro bindings that is not present in the expected bindings", value.Repr(), k)
		}
	}
}
