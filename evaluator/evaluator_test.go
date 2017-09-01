package evaluator

import (
	"math"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
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
		val, ok := (*frame)[e.Identifier(c.identifier)]
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

		call := func(args e.List) (e.Expr, error) {
			funExpr, ok := head(args).(e.Function)
			if !ok {
				t.Errorf("Given %s. Expected %s to be a Function", c.in, funExpr.Repr())
			}
			fun := funExpr.Fun
			return (*fun)(c.args)
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
	ghoul := &Evaluator{env, nil}
	var maxConts float64 = 0
	var maxScopes float64 = 0
	calls := 0

	RegisterFuncAs("checkSize", func(args e.List) (e.Expr, error) {
		calls++

		maxConts = math.Max(float64(len(*((*ghoul).conts))), maxConts)
		maxScopes = math.Max(float64(len(*((*ghoul).env))), maxScopes)
		return head(args), nil
	}, env)

	RegisterFuncAs("eq?", func(args e.List) (e.Expr, error) {
		fst := head(args)
		t, _ := tail(args)
		snd := head(t)
		return e.Boolean(fst.Equiv(snd)), nil
	}, env)

	RegisterFuncAs("+", func(args e.List) (e.Expr, error) {
		fst := head(args).(e.Integer)
		t, _ := tail(args)
		snd := head(t).(e.Integer)
		return e.Integer(fst + snd), nil
	}, env)

	res, err := ghoul.Evaluate(parsed.Expressions)
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
