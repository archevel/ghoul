package evaluator

import (
	"math"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
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
			funExpr, ok := args.Head().(Function)
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
		return args.Head(), nil
	}, env)

	RegisterFuncAs("eq?", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.Head()
		t, _ := args.Tail()
		snd := t.Head()
		return e.Boolean(fst.Equiv(snd)), nil
	}, env)

	RegisterFuncAs("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		fst := args.Head().(e.Integer)
		t, _ := args.Tail()
		snd := t.Head().(e.Integer)
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
	in := `(define-syntax add-one (syntax-rules () (((add-one x) (+ x 1))))) (add-one 5)`
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
(define-syntax my-or (syntax-rules () (((my-or a b) (begin (define tmp a) (cond (tmp tmp) (else b)))))))
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
