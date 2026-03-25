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

		macro := Macro{Pattern: pattern.Expressions.First()}

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

		macro := Macro{Pattern: pattern.Expressions.First()}

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
		{"foo", "foo", newBindings()},
		{"(bar)", "(bar)", newBindings()},
		{"(baz 1)", "(baz x)", b(e.Identifier("x"), e.Integer(1))},
		{"(baz 1 `foo`)", "(baz x y)", b(e.Identifier("x"), e.Integer(1), e.Identifier("y"), e.String("foo"))},
		{"(zoom 1 (love 'foo))", "(zoom x (zoomer z))", b(
			e.Identifier("x"), e.Integer(1),
			e.Identifier("zoomer"), e.Identifier("love"),
			e.Identifier("z"), e.Quote{e.Identifier("foo")},
		)},
		{"(numbers 1 2 3)", "(numbers x y z)", b(
			e.Identifier("x"), e.Integer(1),
			e.Identifier("y"), e.Integer(2),
			e.Identifier("z"), e.Integer(3),
		)},
		{"(numbers 1 2 . 3)", "(numbers x y z)", b(
			e.Identifier("x"), e.Integer(1),
			e.Identifier("y"), e.Integer(2),
			e.Identifier("z"), e.Integer(3),
		)},
		{"(numbers 1 2 3)", "(numbers . x)", b(
			e.Identifier("x"), e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Cons(e.Integer(3), e.NIL))),
		)},
		{"(numbers 1 2 . 3)", "(numbers . x)", b(
			e.Identifier("x"), e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Integer(3))),
		)},
		{"(numbers 1 2 3)", "(numbers x . y)", b(
			e.Identifier("x"), e.Integer(1),
			e.Identifier("y"), e.Cons(e.Integer(2), e.Cons(e.Integer(3), e.NIL)),
		)},
		{"(numbers 1 2 . 3)", "(numbers x . y)", b(
			e.Identifier("x"), e.Integer(1),
			e.Identifier("y"), e.Cons(e.Integer(2), e.Integer(3)),
		)},
		{"(numbers 1 2 3)", "(numbers x y . z)", b(
			e.Identifier("x"), e.Integer(1),
			e.Identifier("y"), e.Integer(2),
			e.Identifier("z"), e.Cons(e.Integer(3), e.NIL),
		)},
		{"(numbers 1 2 . 3)", "(numbers x y . z)", b(
			e.Identifier("x"), e.Integer(1),
			e.Identifier("y"), e.Integer(2),
			e.Identifier("z"), e.Integer(3),
		)},
		{"(numbers 1 2 3)", "(numbers x y z . å)", b(
			e.Identifier("x"), e.Integer(1),
			e.Identifier("y"), e.Integer(2),
			e.Identifier("z"), e.Integer(3),
			e.Identifier("å"), e.NIL,
		)},
		{"(define (love foo za ba) foo bar 1)", "(define (f . a_1) . a_2)", b(
			e.Identifier("f"), e.Identifier("love"),
			e.Identifier("a_1"), e.Cons(e.Identifier("foo"), e.Cons(e.Identifier("za"), e.Cons(e.Identifier("ba"), e.NIL))),
			e.Identifier("a_2"), e.Cons(e.Identifier("foo"), e.Cons(e.Identifier("bar"), e.Cons(e.Integer(1), e.NIL))),
		)},
	}

	for _, c := range cases {
		runBindingTest(t, c.in, c.pattern, c.expectedBindings)
	}
}

func TestMacrosBindCorrectlyWithEllipsisPattern(t *testing.T) {
	// These tests use the old `...` key semantics for patterns where `...`
	// appears as the head (not preceded by a subpattern). These patterns
	// still use the old matchEllipsis path.
	cases := []struct {
		in               string
		pattern          string
		expectedBindings bindings
	}{
		{"(numbers 1 2 3)", "(numbers ...)", b(
			e.Identifier("..."), e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Cons(e.Integer(3), e.NIL))),
		)},
		{"(numbers 1 2 . 3)", "(numbers ...)", b(
			e.Identifier("..."), e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Integer(3))),
		)},
	}

	for _, c := range cases {
		runBindingTest(t, c.in, c.pattern, c.expectedBindings)
	}
}

func TestNestedEllipsisPatternMatching(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		pattern string
		check   func(t *testing.T, result bindings)
	}{
		{
			name:    "let-style: ((var val) ...) body ...",
			in:      "(let ((x 1) (y 2)) (+ x y))",
			pattern: "(let ((var val) ...) body ...)",
			check: func(t *testing.T, result bindings) {
				varVals := result.repeated[e.Identifier("var")]
				if len(varVals) != 2 {
					t.Fatalf("expected 2 var bindings, got %d", len(varVals))
				}
				if !varVals[0].Equiv(e.Identifier("x")) || !varVals[1].Equiv(e.Identifier("y")) {
					t.Errorf("expected var=[x, y], got [%s, %s]", varVals[0].Repr(), varVals[1].Repr())
				}
				valVals := result.repeated[e.Identifier("val")]
				if len(valVals) != 2 {
					t.Fatalf("expected 2 val bindings, got %d", len(valVals))
				}
				if !valVals[0].Equiv(e.Integer(1)) || !valVals[1].Equiv(e.Integer(2)) {
					t.Errorf("expected val=[1, 2], got [%s, %s]", valVals[0].Repr(), valVals[1].Repr())
				}
				bodyVals := result.repeated[e.Identifier("body")]
				if len(bodyVals) != 1 {
					t.Fatalf("expected 1 body binding, got %d", len(bodyVals))
				}
			},
		},
		{
			name:    "empty repetition: ((var val) ...)",
			in:      "(let () (+ 1 2))",
			pattern: "(let ((var val) ...) body ...)",
			check: func(t *testing.T, result bindings) {
				if len(result.repeated[e.Identifier("var")]) != 0 {
					t.Error("expected 0 var bindings for empty binding list")
				}
				if len(result.repeated[e.Identifier("val")]) != 0 {
					t.Error("expected 0 val bindings for empty binding list")
				}
				bodyVals := result.repeated[e.Identifier("body")]
				if len(bodyVals) != 1 {
					t.Fatalf("expected 1 body binding, got %d", len(bodyVals))
				}
			},
		},
		{
			name:    "flat ellipsis: x ...",
			in:      "(my-begin 1 2 3)",
			pattern: "(my-begin x ...)",
			check: func(t *testing.T, result bindings) {
				xVals := result.repeated[e.Identifier("x")]
				if len(xVals) != 3 {
					t.Fatalf("expected 3 x bindings, got %d", len(xVals))
				}
				if !xVals[0].Equiv(e.Integer(1)) || !xVals[1].Equiv(e.Integer(2)) || !xVals[2].Equiv(e.Integer(3)) {
					t.Errorf("expected x=[1,2,3], got [%s,%s,%s]", xVals[0].Repr(), xVals[1].Repr(), xVals[2].Repr())
				}
			},
		},
		{
			name:    "flat ellipsis zero repetitions",
			in:      "(my-begin)",
			pattern: "(my-begin x ...)",
			check: func(t *testing.T, result bindings) {
				xVals := result.repeated[e.Identifier("x")]
				if len(xVals) != 0 {
					t.Fatalf("expected 0 x bindings, got %d", len(xVals))
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
			if patternOk != 0 {
				t.Fatalf("Failed to parse pattern: %s", c.pattern)
			}
			codeOk, code := parser.Parse(strings.NewReader(c.in))
			if codeOk != 0 {
				t.Fatalf("Failed to parse code: %s", c.in)
			}

			pat := pattern.Expressions.First()
			macro := Macro{
				Pattern:      pat,
				PatternVars:  ExtractPatternVars(pat, nil),
				EllipsisVars: ExtractEllipsisVars(pat, nil),
			}

			ok, result := macro.Matches(code.Expressions.First())
			if !ok {
				t.Fatalf("Macro %s did not match %s", c.pattern, c.in)
			}
			c.check(t, result)
		})
	}
}

func TestWildcardPatternMatching(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		pattern string
		check   func(t *testing.T, result bindings)
	}{
		{
			name:    "wildcard matches anything",
			in:      "(mac 42)",
			pattern: "(mac _)",
			check: func(t *testing.T, result bindings) {
				if result.vars[e.Identifier("_")] != nil {
					t.Error("_ should not create a binding")
				}
			},
		},
		{
			name:    "wildcard can appear multiple times",
			in:      "(mac 1 2 3)",
			pattern: "(mac _ x _)",
			check: func(t *testing.T, result bindings) {
				if !result.vars[e.Identifier("x")].Equiv(e.Integer(2)) {
					t.Errorf("expected x=2, got %s", result.vars[e.Identifier("x")].Repr())
				}
				if result.vars[e.Identifier("_")] != nil {
					t.Error("_ should not create a binding")
				}
			},
		},
		{
			name:    "wildcard in nested position",
			in:      "(mac (1 2) 3)",
			pattern: "(mac (_ x) _)",
			check: func(t *testing.T, result bindings) {
				if !result.vars[e.Identifier("x")].Equiv(e.Integer(2)) {
					t.Errorf("expected x=2, got %s", result.vars[e.Identifier("x")].Repr())
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			patternOk, pattern := parser.Parse(strings.NewReader(c.pattern))
			if patternOk != 0 {
				t.Fatalf("Failed to parse pattern: %s", c.pattern)
			}
			codeOk, code := parser.Parse(strings.NewReader(c.in))
			if codeOk != 0 {
				t.Fatalf("Failed to parse code: %s", c.in)
			}
			macro := Macro{Pattern: pattern.Expressions.First()}
			ok, result := macro.Matches(code.Expressions.First())
			if !ok {
				t.Fatalf("Macro %s did not match %s", c.pattern, c.in)
			}
			c.check(t, result)
		})
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

		macro := Macro{Pattern: pattern.Expressions.First()}

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
		{"foo", "foo", newBindings()},
		{"(bar)", "(bar)", newBindings()},

		{"(baz 1)", "(baz x)", b(e.Identifier("x"), e.Integer(1))},
		{"(baz 1 \"foo\")", "(baz x y)", b(e.Identifier("x"), e.Integer(1), e.Identifier("y"), e.String("foo"))},
	}

	for _, c := range cases {

		bodyOk, body := parser.Parse(strings.NewReader(c.body))
		if bodyOk != 0 {
			t.Fatal("Parsing pattern failed")
		}

		macro := Macro{Body: body.Expressions.First()}

		expanded := macro.ExpandHygienic(c.bound, 0)

		if expanded.Repr() != c.expectedRepr {
			t.Errorf("Expected %s after expanding macro, but got %s", c.expectedRepr, expanded.Repr())
		}
	}
}

func swapMacroExample() {

	_, pattern := parser.Parse(strings.NewReader("(swap x y)"))
	_, body := parser.Parse(strings.NewReader("(let ((tmp x)) (set! x y) (set! y tmp))"))
	_, code := parser.Parse(strings.NewReader("(swap foo bar)"))

	macro := Macro{Pattern: pattern.Expressions.(e.List).First(), Body: body.Expressions.(e.List).First()}
	_, bound := macro.Matches(code.Expressions.(e.List).First())

	res := macro.ExpandHygienic(bound, 0)
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

		macro := Macro{Pattern: pattern.Expressions.First(), Body: body.Expressions.First()}

		codeOk, code := parser.Parse(strings.NewReader(c.in))

		if codeOk != 0 {
			t.Fatal("Parsing code failed")
		}

		bindOk, bound := macro.Matches(code.Expressions.(e.List).First())

		if !bindOk {
			t.Errorf("Could not bind %s to patterns in %s", c.in, c.pattern)
		}

		res := macro.ExpandHygienic(bound, 0)

		if res.Repr() != c.out {
			t.Errorf("Expansion of %s did not give expected result %s, instead got %+v", c.in, c.out, res.Repr())
		}
	}
}

// b creates a bindings struct with only single (non-repeated) vars for test convenience.
func b(pairs ...interface{}) bindings {
	result := newBindings()
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i].(e.Identifier)
		val := pairs[i+1].(e.Expr)
		result.vars[key] = val
	}
	return result
}

func runBindingTest(t *testing.T, in string, patternStr string, bound bindings) {

	patternOk, pattern := parser.Parse(strings.NewReader(patternStr))
	if patternOk != 0 {
		t.Fatalf("Parsing pattern '%s' failed", pattern)
	}

	macro := Macro{Pattern: pattern.Expressions.First()}

	parseOk, parseRes := parser.Parse(strings.NewReader(in))

	if parseOk != 0 {
		t.Fatalf("Parsing code %s failed", in)
	}
	_, result := macro.Matches(parseRes.Expressions.First())
	if len(result.vars) != len(bound.vars) {
		t.Errorf(`Macro %s did not bind correctly for %s. Expected %d bindings got %d`,
			patternStr, in, len(bound.vars), len(result.vars))
	}

	for k, expectedValue := range bound.vars {
		value := result.vars[k]
		if value == nil {
			t.Errorf("Expected value %s for key %s in %s using %s, but got nil!", expectedValue.Repr(), k.Repr(), in, patternStr)
		} else if !expectedValue.Equiv(value) {
			t.Errorf("Expected value %s for key %s in %s using %s in bindings, got %s",
				expectedValue.Repr(), k.Repr(), in, patternStr, value.Repr())
		}
	}

	for k, value := range result.vars {
		if !value.Equiv(bound.vars[k]) {
			t.Errorf("Found value %s for key %s in macro bindings that is not present in the expected bindings", value.Repr(), k)
		}
	}
}
