package evaluator

import (
	"fmt"
	"math"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
	"github.com/archevel/ghoul/macromancy"
	p "github.com/archevel/ghoul/parser"
)

func TestEvaluatesSimpleValues(t *testing.T) {

	cases := []struct {
		in  string
		out e.Expr
	}{
		{"1", e.Integer(1)},
		{"", e.NIL},
		{"\n", e.NIL},
		{"`foo`", e.String("foo")},
		{"2.01", e.Float(2.01)},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestEvaluatesToLastExpression(t *testing.T) {

	cases := []struct {
		in  string
		out e.Expr
	}{
		{"1\n2", e.Integer(2)},
		{"1\n", e.Integer(1)},
		{"`foo`\n\"bar\"", e.String("bar")},
		{"2.0 33.1", e.Float(33.1)},
		{"'a\n\n'(99 1)", e.Cons(e.Integer(99), e.Cons(e.Integer(1), e.NIL))},
		{"'foo 'mmm", e.Identifier("mmm")},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestVariableDefinitionsAreStoredInEnvironment(t *testing.T) {
	env := NewEnvironment()

	cases := []struct {
		in         string
		identifier string
		out        e.Expr
	}{
		{`(define x 1)`, "x", e.Integer(1)},
		{`(define z "love")`, "z", e.String("love")},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		_, parsed := p.Parse(r)

		Evaluate(parsed.Expressions, env)

		frame := currentScope(env)
		val, ok := (*frame)[keyFromIdentifier(e.Identifier(c.identifier))]
		if !ok {
			t.Errorf("environment had no value for '%s'", c.identifier)
		}

		if val != c.out {

			t.Errorf("Expected environment to hold %v for  %s, butwas %v", c.out.Repr(), c.identifier, val.Repr())
		}
	}
}

func TestVariablesValuesInEnvironmentReplaceTheirIdentifiers(t *testing.T) {
	cases := []struct {
		in  string
		out e.Expr
	}{
		{`(define x 1) x`, e.Integer(1)},
		{`(define z "love") z`, e.String("love")},
		{`(define fool 'a_fool) fool`, e.Identifier("a_fool")},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestDefineEvaluatesTheValue(t *testing.T) {
	cases := []struct {
		in  string
		out e.Expr
	}{
		{`(define x 1) (define y x) y`, e.Integer(1)},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestLambdasAreMonadicFunctions(t *testing.T) {
	env := NewEnvironment()
	cases := []struct {
		in   string
		args e.List
		out  e.Expr
	}{
		{`(call (lambda () 1))`, e.NIL, e.Integer(1)},
		{`(call (lambda () 1 2 3 "four"))`, e.NIL, e.String("four")},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		_, parsed := p.Parse(r)

		call := func(args e.List, ev *Evaluator) (e.Expr, error) {
			funExpr, ok := args.First().(Function)
			if !ok {
				t.Errorf("Given %s. Expected %s to be a Function", c.in, funExpr.Repr())
			}
			fun := funExpr.Fun
			return (*fun)(c.args, ev)
		}
		RegisterFuncAs("call", call, env)
		res, _ := Evaluate(parsed.Expressions, env)

		if res != c.out {
			t.Errorf("Given %s, Expected call to it to give %v, but got %v", c.in, c.out.Repr(), res.Repr())
		}

	}
}

func TestLambdasAreCallable(t *testing.T) {

	in := `((lambda () 1))`
	out := e.Integer(1)

	testInputGivesOutput(in, out, t)
}

func TestLambdasBindCallArgsToParams(t *testing.T) {

	in := `((lambda (x) x) 'an-arg)`
	out := e.Identifier("an-arg")

	testInputGivesOutput(in, out, t)
}

func TestLambdasBindMultipleArgsToParams(t *testing.T) {

	in := `((lambda (x y z) y) 123 678 'an-arg)`
	out := e.Integer(678)

	testInputGivesOutput(in, out, t)
}

func TestLambdasCanAcceptingVariadicArguments(t *testing.T) {

	cases := []struct {
		in  string
		out e.Expr
	}{
		{`((lambda (x . y) y) 1 2 3 4)`, list(e.Integer(2), e.Integer(3), e.Integer(4))},
		{`((lambda y y) 4 99)`, list(e.Integer(4), e.Integer(99))},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}

}

func TestLambdaBodiesAreEvaluatedWhenCalled(t *testing.T) {

	in := `((lambda () ((lambda () "foo"))))`
	out := e.String("foo")

	testInputGivesOutput(in, out, t)
}

func TestLambdasHaveAccessToOuterScope(t *testing.T) {

	in := `((lambda (x) ((lambda (y) x) 99)) 11)`
	out := e.Integer(11)

	testInputGivesOutput(in, out, t)
}

func TestLambdasDefinedVariablesRevertToTheirOriginalValue(t *testing.T) {

	in := `(define x 77) ((lambda () (define x 'foo) 33)) x`
	out := e.Integer(77)
	testInputGivesOutput(in, out, t)

}

func TestQuoteYieldsQuoted(t *testing.T) {
	cases := []struct {
		in  string
		out e.Expr
	}{
		{"'()", e.NIL},
		{"''()", e.Quote{e.NIL}},
		{"'42", e.Integer(42)},
		{"'42.7", e.Float(42.7)},
		{"'a", e.Identifier("a")},
		{"'`a raw string`", e.String("a raw string")},
		{"'(99 . 1)", e.Cons(e.Integer(99), e.Integer(1))},
		{"'foo", e.Identifier("foo")},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestCondYieldsTruthyPathOrNIL(t *testing.T) {
	cases := []struct {
		in  string
		out e.Expr
	}{
		// predicate true
		{`(cond (#t 1))`, e.Integer(1)},
		{`(cond ('(123) 1))`, e.Integer(1)},
		//		{`(cond ((lambda () #f) 1))`, e.Integer(1)},
		{`(cond (0 1))`, e.Integer(1)},
		{`(cond ("" 1))`, e.Integer(1)},
		{`(cond ("false" 1))`, e.Integer(1)},
		{`(cond ('baz 1))`, e.Integer(1)},

		// nil when no true predicate
		{`(cond)`, e.NIL},
		{`(cond (((lambda () #f)) 1))`, e.NIL},
		{`(cond (#f 1))`, e.NIL},
		{`(cond ('() 1))`, e.NIL},
		{`(cond ('() 1) (#f 2) (((lambda () #f)) 3))`, e.NIL},

		// first predicate false
		{`(cond (((lambda () #f)) 1) (2 2))`, e.Integer(2)},
		{`(cond (#f 1) ('2 2))`, e.Integer(2)},
		{`(cond ('() 1) ("two" 2))`, e.Integer(2)},

		// with else
		{`(cond (else 1))`, e.Integer(1)},
		{`(cond (else 0) ('baz 1))`, e.Integer(0)},
		{`(cond (#f 1) (else 2))`, e.Integer(2)},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestBeginYieldsLastValue(t *testing.T) {
	cases := []struct {
		in  string
		out e.Expr
	}{
		{"(begin `foo`)", e.String("foo")},
		{"(define x (begin 1 2 3)) x", e.Integer(3)},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)

		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestAssignmentWorksWhenVariableIsDefined(t *testing.T) {
	in := `(define x "foo") (set! x 5) x`
	out := e.Integer(5)
	testInputGivesOutput(in, out, t)
}

func TestAssignmentEvaluatesValue(t *testing.T) {
	in := `(define x "foo") (set! x (begin 1 2 3)) x`
	out := e.Integer(3)
	testInputGivesOutput(in, out, t)
}

func TestAssignmentOnlyChangesWithinTheSmallestScope(t *testing.T) {

	cases := []struct {
		in  string
		out e.Expr
	}{
		{`(define x "foo") ((lambda () (define x "baz") (set! x 5) x)) x`, e.String("foo")},

		{`(define x "foo") ((lambda (x) (set! x 5) x) x)`, e.Integer(5)},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestContextGrowthOnTailRecursiveCall(t *testing.T) {

	in := `
	(define foo (lambda (n) 
		(checkSize n)
		(cond ((eq? n 100) n) 
		(else (begin (foo (+ n 1)))))))
	(foo 0)	
	`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)

	env := NewEnvironment()
	evaluator := New(logging.NoLogger, env)

	var maxConts float64 = 0
	var maxScopes float64 = 0
	calls := 0

	RegisterFuncAs("checkSize", func(args e.List, ev *Evaluator) (e.Expr, error) {
		calls++

		maxConts = math.Max(float64(len(*((*evaluator).conts))), maxConts)
		maxScopes = math.Max(float64(len(*((*evaluator).env))), maxScopes)
		return args.First(), nil
	}, env)

	RegisterFuncAs("eq?", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First()
		t, _ := args.Tail()
		snd := t.First()
		return e.Boolean(fst.Equiv(snd)), nil
	}, env)

	RegisterFuncAs("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Integer(fst + snd), nil
	}, env)

	res, err := evaluator.Evaluate(parsed.Expressions)
	out := e.Integer(100)

	if err != nil {
		t.Error("Got evaluation error", err)
	}
	if !out.Equiv(res) {
		t.Errorf("Expected %+v, but got %+v", out.Repr(), res.Repr())
	}
	if maxConts != 3.0 {
		t.Errorf("Bad maxConts: %v", maxConts)
	}

	if maxScopes != 2.0 {
		t.Errorf("Bad maxScopes %v", maxScopes)
	}
}

func TestDefineSyntaxWithSyntaxRules(t *testing.T) {
	in := `(define-syntax add-one (syntax-rules () ((add-one x) (+ x 1)))) (add-one 5)`
	env := NewEnvironment()
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t2, _ := args.Tail()
		snd := t2.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})
	testInputGivesOutputWithinEnv(in, e.Integer(6), env, t)
}

func TestDefineSyntaxHygiene(t *testing.T) {
	// The classic hygiene test: macro introduces "tmp", user also has "tmp"
	in := `
(define-syntax my-or (syntax-rules () ((my-or a b) (begin (define tmp a) (cond (tmp tmp) (else b))))))
(define tmp 5)
(my-or #f tmp)
`
	env := NewEnvironment()
	testInputGivesOutputWithinEnv(in, e.Integer(5), env, t)
}

func TestDefineSyntaxWithLambdaTransformer(t *testing.T) {
	// A general macro transformer using a lambda
	// The lambda receives the whole form as first arg and should return the expansion
	in := `
(define-syntax double
  (lambda (stx)
    (begin
      (define val (+ 0 0))
      val)))
(double 5)
`
	env := NewEnvironment()
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})
	testInputGivesOutputWithinEnv(in, e.Integer(0), env, t)
}

func TestGeneralTransformerCanReturnArbitraryCode(t *testing.T) {
	// A general transformer that ignores input and returns a literal
	in := `
(define-syntax always-42
  (lambda (stx) 42))
(always-42 anything here)
`
	env := NewEnvironment()
	testInputGivesOutputWithinEnv(in, e.Integer(42), env, t)
}

func TestDefineSyntaxErrorCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"missing name and transformer", `(define-syntax)`},
		{"non-identifier name", `(define-syntax 42 (syntax-rules () (foo bar)))`},
		{"missing transformer", `(define-syntax foo)`},
		{"non-list transformer", `(define-syntax foo bar)`},
		{"non-procedure general transformer", `(define-syntax foo (begin 42))`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := NewEnvironment()
			r := strings.NewReader(c.in)
			_, parsed := p.Parse(r)
			_, err := Evaluate(parsed.Expressions, env)
			if err == nil {
				t.Errorf("expected error for '%s'", c.in)
			}
		})
	}
}

func TestSyntaxRulesNoMatchingPattern(t *testing.T) {
	in := `(define-syntax foo (syntax-rules () ((foo x y) (+ x y)))) (foo 1)`
	env := NewEnvironment()
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Error("expected error for arity mismatch")
	}
}

func TestScopedIdentifierMacroCallExpansion(t *testing.T) {
	// Test that a ScopedIdentifier resolving to a SyntaxTransformer works
	env := NewEnvironment()
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})

	in := `(define-syntax add1 (syntax-rules () ((add1 x) (+ x 1)))) (add1 5)`
	testInputGivesOutputWithinEnv(in, e.Integer(6), env, t)
}

func TestLookupFallbackForMarkedIdentifiers(t *testing.T) {
	env := NewEnvironment()
	bindIdentifier(e.Identifier("x"), e.Integer(42), env)

	// A ScopedIdentifier with marks should fall back to unmarked binding
	si := e.ScopedIdentifier{Name: e.Identifier("x"), Marks: map[uint64]bool{99: true}}
	result, err := lookupIdentifier(si, env)
	if err != nil {
		t.Fatalf("expected fallback lookup to succeed: %v", err)
	}
	if !result.Equiv(e.Integer(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestLookupExactMatchTakesPriorityOverFallback(t *testing.T) {
	env := NewEnvironment()
	bindIdentifier(e.Identifier("x"), e.Integer(1), env)
	si := e.ScopedIdentifier{Name: e.Identifier("x"), Marks: map[uint64]bool{5: true}}
	bindIdentifier(si, e.Integer(2), env)

	// Exact match (name+marks) should win over fallback (name only)
	result, err := lookupIdentifier(si, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.Integer(2)) {
		t.Errorf("expected exact match (2), got %s", result.Repr())
	}
}

func TestLookupFallbackDoesNotApplyToUnmarked(t *testing.T) {
	env := NewEnvironment()
	// Don't bind "y" at all
	_, err := lookupIdentifier(e.Identifier("y"), env)
	if err == nil {
		t.Error("expected error for undefined identifier")
	}
}

func TestResolveCallableHeadWithScopedIdentifier(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})

	// Call (+ 1 2) where + is a ScopedIdentifier — should resolve via fallback
	si := e.ScopedIdentifier{Name: e.Identifier("+"), Marks: map[uint64]bool{1: true}}
	expr := e.Cons(si, e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.NIL)))

	evaluator := New(logging.StandardLogger, env)
	result, err := evaluator.Evaluate(e.Cons(expr, e.NIL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.Integer(3)) {
		t.Errorf("expected 3, got %s", result.Repr())
	}
}

func TestResolveCallableHeadWithScopedSyntaxTransformer(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})

	// Bind a SyntaxTransformer under a ScopedIdentifier
	si := e.ScopedIdentifier{Name: e.Identifier("my-add"), Marks: map[uint64]bool{1: true}}
	transformer := SyntaxTransformer{
		Transform: func(code e.List, mark uint64) (e.Expr, error) {
			// (my-add x y) -> (+ x y)
			tail, _ := code.Tail()
			return e.Cons(e.Identifier("+"), tail), nil
		},
	}
	bindIdentifier(si, transformer, env)

	// Call via the same ScopedIdentifier
	expr := e.Cons(si, e.Cons(e.Integer(3), e.Cons(e.Integer(4), e.NIL)))

	evaluator := New(logging.StandardLogger, env)
	result, err := evaluator.Evaluate(e.Cons(expr, e.NIL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.Integer(7)) {
		t.Errorf("expected 7, got %s", result.Repr())
	}
}

func TestResolveCallableHeadWithNonIdentifier(t *testing.T) {
	env := NewEnvironment()
	// A list whose head is an integer — not resolvable as a transformer,
	// falls through to normal function call which will fail
	expr := e.Cons(e.Cons(e.Integer(1), e.NIL), e.NIL)

	evaluator := New(logging.StandardLogger, env)
	_, err := evaluator.Evaluate(e.Cons(expr, e.NIL))
	if err == nil {
		t.Error("expected error when calling a non-procedure")
	}
}

func TestSyntaxTransformerReprAndEquiv(t *testing.T) {
	st := SyntaxTransformer{Transform: func(code e.List, mark uint64) (e.Expr, error) { return e.NIL, nil }}
	if st.Repr() != "#<syntax-transformer>" {
		t.Errorf("expected '#<syntax-transformer>', got '%s'", st.Repr())
	}
	if st.Equiv(st) {
		t.Error("SyntaxTransformer should never be Equiv to anything")
	}
	if st.Equiv(e.Integer(1)) {
		t.Error("SyntaxTransformer should never be Equiv to anything")
	}
}

func TestGeneralSyntaxTransformerReprAndEquiv(t *testing.T) {
	gst := GeneralSyntaxTransformer{}
	if gst.Repr() != "#<general-syntax-transformer>" {
		t.Errorf("expected '#<general-syntax-transformer>', got '%s'", gst.Repr())
	}
	if gst.Equiv(gst) {
		t.Error("GeneralSyntaxTransformer should never be Equiv to anything")
	}
}

func TestScopedIdentifierLookupDuringEvaluation(t *testing.T) {
	env := NewEnvironment()

	// Bind a ScopedIdentifier
	si := e.ScopedIdentifier{Name: e.Identifier("x"), Marks: map[uint64]bool{1: true}}
	bindIdentifier(si, e.Integer(42), env)

	// Evaluate an expression tree that references the ScopedIdentifier
	// (define y <scoped-x>) y
	expr := e.Cons(
		e.Cons(e.Identifier("define"), e.Cons(e.Identifier("y"), e.Cons(si, e.NIL))),
		e.Cons(e.Identifier("y"), e.NIL),
	)

	evaluator := New(logging.StandardLogger, env)
	result, err := evaluator.Evaluate(expr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.Integer(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestScopedIdentifierAndPlainIdentifierAreDistinct(t *testing.T) {
	env := NewEnvironment()

	// Bind plain "x" = 1
	bindIdentifier(e.Identifier("x"), e.Integer(1), env)
	// Bind scoped "x" with mark = 2
	si := e.ScopedIdentifier{Name: e.Identifier("x"), Marks: map[uint64]bool{1: true}}
	bindIdentifier(si, e.Integer(2), env)

	// Evaluate plain x — should get 1
	expr1 := e.Cons(e.Identifier("x"), e.NIL)
	evaluator1 := New(logging.StandardLogger, env)
	result1, err := evaluator1.Evaluate(expr1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result1.Equiv(e.Integer(1)) {
		t.Errorf("plain x should be 1, got %s", result1.Repr())
	}

	// Evaluate scoped x — should get 2
	expr2 := e.Cons(si, e.NIL)
	evaluator2 := New(logging.StandardLogger, env)
	result2, err := evaluator2.Evaluate(expr2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result2.Equiv(e.Integer(2)) {
		t.Errorf("scoped x should be 2, got %s", result2.Repr())
	}
}

func TestGeneralTransformerDefineInsideLetInsideNestedCall(t *testing.T) {
	// When a general transformer uses `let` (a syntax-rules macro) and
	// that transformer is called from within another general transformer,
	// the `let` bindings should resolve correctly. Previously, the double
	// hygiene marking (outer transformer + inner transformer) caused
	// identifiers defined via `let` to become unreachable.
	in := `
(define-syntax let (syntax-rules ()
  ((let ((var val) ...) body ...)
   ((lambda (var ...) body ...) val ...))))

(define-syntax inner-mac
  (lambda (stx)
    (let ((x 42)) x)))

(define-syntax outer-mac
  (lambda (stx)
    (inner-mac)))

(outer-mac)
`
	testInputGivesOutput(in, e.Integer(42), t)
}

func TestGeneralTransformerNestedDefineInsideLambda(t *testing.T) {
	// When a general transformer defines a function inside a lambda
	// (nested define), and that inner function uses car/cdr on the
	// transformer's syntax input, it should see the same data as
	// the outer scope. This fails when the inner scope's environment
	// causes different identifier resolution.
	env := NewEnvironment()
	env.Register("cdr", func(args e.List, ev *Evaluator) (e.Expr, error) {
		arg := args.First()
		if list, ok := arg.(e.List); ok && list != e.NIL {
			return list.Second(), nil
		}
		return e.NIL, fmt.Errorf("cdr: expected list, got %T", arg)
	})
	env.Register("car", func(args e.List, ev *Evaluator) (e.Expr, error) {
		if list, ok := args.First().(e.List); ok && list != e.NIL {
			return list.First(), nil
		}
		return e.NIL, fmt.Errorf("car: expected list, got %T", args.First())
	})
	env.Register("identifier?", func(args e.List, evaluator *Evaluator) (e.Expr, error) {
		arg := args.First()
		if so, ok := arg.(macromancy.SyntaxObject); ok {
			_, isId := so.Datum.(e.Identifier)
			_, isSI := so.Datum.(e.ScopedIdentifier)
			return e.Boolean(isId || isSI), nil
		}
		_, isId := arg.(e.Identifier)
		_, isSI := arg.(e.ScopedIdentifier)
		return e.Boolean(isId || isSI), nil
	})
	env.Register("syntax->datum", func(args e.List, evaluator *Evaluator) (e.Expr, error) {
		if so, ok := args.First().(macromancy.SyntaxObject); ok {
			if si, ok := so.Datum.(e.ScopedIdentifier); ok {
				return si.Name, nil
			}
			return so.Datum, nil
		}
		if si, ok := args.First().(e.ScopedIdentifier); ok {
			return si.Name, nil
		}
		return args.First(), nil
	})
	env.Register("null?", func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.Boolean(args.First() == e.NIL), nil
	})
	env.Register("pair?", func(args e.List, ev *Evaluator) (e.Expr, error) {
		_, ok := args.First().(*e.Pair)
		return e.Boolean(ok), nil
	})
	env.Register("cons", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First()
		t2, _ := args.Tail()
		snd := t2.First()
		return e.Cons(fst, snd), nil
	})

	in := `
(define-syntax test-mac
  (lambda (stx)
    (define clauses (cdr (cdr stx)))
    (define clause (car clauses))
    (define pat (car clause))
    (define collect
      (lambda (p)
        (define walk
          (lambda (expr acc)
            (cond
              ((null? expr) acc)
              ((identifier? expr) (+ acc 1))
              ((pair? expr) (walk (cdr expr) (walk (car expr) acc)))
              (else acc))))
        (walk (cdr p) 0)))
    (collect pat)))
(test-mac ()
  ((test-mac x) 42))
`
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t2, _ := args.Tail()
		snd := t2.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	result, err := Evaluate(parsed.Expressions, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return 1 — one pattern variable 'x' in pattern (test-mac x)
	if !result.Equiv(e.Integer(1)) {
		t.Errorf("expected 1, got %s", result.Repr())
	}
}

func TestGeneralTransformerRecursiveFnInsideLetInNestedCall(t *testing.T) {
	// A recursive function defined inside a general transformer that uses
	// `let`, called from another transformer. This is the pattern needed
	// for syntax-case as a prelude macro.
	in := `
(define-syntax let (syntax-rules ()
  ((let ((var val) ...) body ...)
   ((lambda (var ...) body ...) val ...))))

(define-syntax count-mac
  (lambda (stx)
    (define count
      (lambda (lst acc)
        (cond
          ((null? lst) acc)
          (else (count (cdr lst) (+ acc 1))))))
    (let ((result (count (cdr stx) 0)))
      result)))

(define-syntax outer
  (lambda (stx)
    (count-mac a b c)))

(outer)
`
	env := NewEnvironment()
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})
	env.Register("null?", func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.Boolean(args.First() == e.NIL), nil
	})
	env.Register("cdr", func(args e.List, ev *Evaluator) (e.Expr, error) {
		if list, ok := args.First().(e.List); ok && list != e.NIL {
			return list.Second(), nil
		}
		return e.NIL, nil
	})
	testInputGivesOutputWithinEnv(in, e.Integer(3), env, t)
}

func testInputGivesOutput(in string, out e.Expr, t *testing.T) {
	env := NewEnvironment()
	testInputGivesOutputWithinEnv(in, out, env, t)
}

func testInputGivesOutputWithinEnv(in string, out e.Expr, env *environment, t *testing.T) {
	r := strings.NewReader(in)
	parseRes, parsed := p.Parse(r)

	if parseRes != 0 {
		t.Errorf("Parser failed given: %s", in)
	}

	res, err := Evaluate(parsed.Expressions, env)

	if err != nil {
		t.Errorf("Given %s. Got unexpected error: %q", in, err)
	}

	if res == nil {
		t.Errorf("Given %s. Resulted in 'nil', expected %s", in, out.Repr())
	} else if !out.Equiv(res) {
		t.Errorf("Given %s. Expected %s to be equivalent to %s", in, res.Repr(), out.Repr())
	}
}
