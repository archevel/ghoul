package consume

import (
	"context"
	"math"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
	p "github.com/archevel/ghoul/exhumer"
)

// testInputGivesOutput parses and evaluates the input, asserting it produces
// the expected output expression.
func testInputGivesOutput(in string, out *e.Node, t *testing.T) {
	t.Helper()
	env := NewEnvironment()
	testInputGivesOutputWithinEnv(in, out, env, t)
}

func testInputGivesOutputWithinEnv(in string, out *e.Node, env *environment, t *testing.T) {
	t.Helper()
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
	} else if !res.Equiv(out) {
		t.Errorf("Given %s. Expected %s to be equivalent to %s", in, res.Repr(), out.Repr())
	}
}

func TestEvaluatesSimpleValues(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Node
	}{
		{"1", e.IntNode(1)},
		{"", e.Nil},
		{"\n", e.Nil},
		{"`foo`", e.StrNode("foo")},
		{"2.01", e.FloatNode(2.01)},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestEvaluatesToLastExpression(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Node
	}{
		{"1\n2", e.IntNode(2)},
		{"1\n", e.IntNode(1)},
		{"`foo`\n\"bar\"", e.StrNode("bar")},
		{"2.0 33.1", e.FloatNode(33.1)},
		{"'a\n\n'(99 1)", e.NewListNode([]*e.Node{e.IntNode(99), e.IntNode(1)})},
		{"'foo 'mmm", e.IdentNode("mmm")},
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
		out        *e.Node
	}{
		{`(define x 1)`, "x", e.IntNode(1)},
		{`(define z "love")`, "z", e.StrNode("love")},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		_, parsed := p.Parse(r)

		Evaluate(parsed.Expressions, env)

		frame := currentScope(env)
		val, ok := (*frame)[keyFromName(c.identifier)]
		if !ok {
			t.Errorf("environment had no value for '%s'", c.identifier)
		}

		if !val.Equiv(c.out) {
			t.Errorf("Expected environment to hold %v for  %s, but was %v", c.out.Repr(), c.identifier, val.Repr())
		}
	}
}

func TestVariablesValuesInEnvironmentReplaceTheirIdentifiers(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{`(define x 1) x`, e.IntNode(1)},
		{`(define z "love") z`, e.StrNode("love")},
		{`(define fool 'a_fool) fool`, e.IdentNode("a_fool")},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestDefineEvaluatesTheValue(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{`(define x 1) (define y x) y`, e.IntNode(1)},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestLambdasAreMonadicFunctions(t *testing.T) {
	env := NewEnvironment()
	cases := []struct {
		in   string
		args []*e.Node
		out  *e.Node
	}{
		{`(call (lambda () 1))`, nil, e.IntNode(1)},
		{`(call (lambda () 1 2 3 "four"))`, nil, e.StrNode("four")},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		_, parsed := p.Parse(r)

		call := func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
			fn := args[0]
			if fn.Kind != e.FunctionNode || fn.FuncVal == nil {
				t.Errorf("Given %s. Expected FunctionNode, got %s", c.in, fn.Repr())
				return e.Nil, nil
			}
			return (*fn.FuncVal)(c.args, ev)
		}
		RegisterFuncAs("call", call, env)
		res, _ := Evaluate(parsed.Expressions, env)

		if !res.Equiv(c.out) {
			t.Errorf("Given %s, Expected call to it to give %v, but got %v", c.in, c.out.Repr(), res.Repr())
		}

	}
}

func TestLambdasAreCallable(t *testing.T) {

	in := `((lambda () 1))`
	out := e.IntNode(1)

	testInputGivesOutput(in, out, t)
}

func TestLambdasBindCallArgsToParams(t *testing.T) {

	in := `((lambda (x) x) 'an-arg)`
	out := e.IdentNode("an-arg")

	testInputGivesOutput(in, out, t)
}

func TestLambdasBindMultipleArgsToParams(t *testing.T) {

	in := `((lambda (x y z) y) 123 678 'an-arg)`
	out := e.IntNode(678)

	testInputGivesOutput(in, out, t)
}

func TestLambdasCanAcceptingVariadicArguments(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Node
	}{
		{`((lambda (x . y) y) 1 2 3 4)`, e.NewListNode([]*e.Node{e.IntNode(2), e.IntNode(3), e.IntNode(4)})},
		{`((lambda y y) 4 99)`, e.NewListNode([]*e.Node{e.IntNode(4), e.IntNode(99)})},
	}

	for _, c := range cases {
		testInputGivesOutput(c.in, c.out, t)
	}

}

func TestLambdaBodiesAreEvaluatedWhenCalled(t *testing.T) {

	in := `((lambda () ((lambda () "foo"))))`
	out := e.StrNode("foo")

	testInputGivesOutput(in, out, t)
}

func TestLambdasHaveAccessToOuterScope(t *testing.T) {

	in := `((lambda (x) ((lambda (y) x) 99)) 11)`
	out := e.IntNode(11)

	testInputGivesOutput(in, out, t)
}

func TestLambdasDefinedVariablesRevertToTheirOriginalValue(t *testing.T) {

	in := `(define x 77) ((lambda () (define x 'foo) 33)) x`
	out := e.IntNode(77)
	testInputGivesOutput(in, out, t)

}

func TestQuoteYieldsQuoted(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{"'()", e.Nil},
		{"''()", e.QuoteNodeVal(e.Nil)},
		{"'42", e.IntNode(42)},
		{"'42.7", e.FloatNode(42.7)},
		{"'a", e.IdentNode("a")},
		{"'`a raw string`", e.StrNode("a raw string")},
		{"'(99 . 1)", &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(99)}, DottedTail: e.IntNode(1)}},
		{"'foo", e.IdentNode("foo")},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestCondYieldsTruthyPathOrNIL(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		// predicate true
		{`(cond (#t 1))`, e.IntNode(1)},
		{`(cond ('(123) 1))`, e.IntNode(1)},
		//		{`(cond ((lambda () #f) 1))`, e.IntNode(1)},
		{`(cond (0 1))`, e.IntNode(1)},
		{`(cond ("" 1))`, e.IntNode(1)},
		{`(cond ("false" 1))`, e.IntNode(1)},
		{`(cond ('baz 1))`, e.IntNode(1)},

		// nil when no true predicate
		{`(cond)`, e.Nil},
		{`(cond (((lambda () #f)) 1))`, e.Nil},
		{`(cond (#f 1))`, e.Nil},
		{`(cond ('() 1))`, e.Nil},
		{`(cond ('() 1) (#f 2) (((lambda () #f)) 3))`, e.Nil},

		// first predicate false
		{`(cond (((lambda () #f)) 1) (2 2))`, e.IntNode(2)},
		{`(cond (#f 1) ('2 2))`, e.IntNode(2)},
		{`(cond ('() 1) ("two" 2))`, e.IntNode(2)},

		// with else
		{`(cond (else 1))`, e.IntNode(1)},
		{`(cond (else 0) ('baz 1))`, e.IntNode(0)},
		{`(cond (#f 1) (else 2))`, e.IntNode(2)},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestBeginYieldsLastValue(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{"(begin `foo`)", e.StrNode("foo")},
		{"(define x (begin 1 2 3)) x", e.IntNode(3)},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)

		testInputGivesOutput(c.in, c.out, t)
	}
}

func TestAssignmentWorksWhenVariableIsDefined(t *testing.T) {
	in := `(define x "foo") (set! x 5) x`
	out := e.IntNode(5)
	testInputGivesOutput(in, out, t)
}

func TestAssignmentEvaluatesValue(t *testing.T) {
	in := `(define x "foo") (set! x (begin 1 2 3)) x`
	out := e.IntNode(3)
	testInputGivesOutput(in, out, t)
}

func TestAssignmentOnlyChangesWithinTheSmallestScope(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Node
	}{
		{`(define x "foo") ((lambda () (define x "baz") (set! x 5) x)) x`, e.StrNode("foo")},

		{`(define x "foo") ((lambda (x) (set! x 5) x) x)`, e.IntNode(5)},
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
	evaluator := New(engraving.NoLogger, env)

	var maxScopes float64 = 0
	calls := 0

	RegisterFuncAs("checkSize", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		calls++
		maxScopes = math.Max(float64(len(*(ev.env))), maxScopes)
		return args[0], nil
	}, env)

	RegisterFuncAs("eq?", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.BoolNode(args[0].Equiv(args[1])), nil
	}, env)

	RegisterFuncAs("+", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		fst := args[0].IntVal
		snd := args[1].IntVal
		return e.IntNode(fst + snd), nil
	}, env)

	res, err := evaluator.EvaluateNode(context.Background(), parsed.Expressions)

	if err != nil {
		t.Error("Got evaluation error", err)
	}
	if !res.Equiv(e.IntNode(100)) {
		t.Errorf("Expected 100, but got %+v", res.Repr())
	}
	// With the bytecode VM, scope depth should remain bounded for tail calls
	if maxScopes > 3.0 {
		t.Errorf("Bad maxScopes %v", maxScopes)
	}
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
	env.Register("+", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		fst := args[0].IntVal
		snd := args[1].IntVal
		return e.IntNode(fst + snd), nil
	})
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Error("expected error for arity mismatch")
	}
}

func TestLookupFallbackForMarkedIdentifiers(t *testing.T) {
	env := NewEnvironment()
	bindNode(e.IdentNode("x"), e.IntNode(42), env)

	// A ScopedIdentifier with marks should fall back to unmarked binding
	si := e.ScopedIdentNode("x", map[uint64]bool{99: true})
	result, err := lookupNode(si, env)
	if err != nil {
		t.Fatalf("expected fallback lookup to succeed: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestLookupExactMatchTakesPriorityOverFallback(t *testing.T) {
	env := NewEnvironment()
	bindNode(e.IdentNode("x"), e.IntNode(1), env)
	si := e.ScopedIdentNode("x", map[uint64]bool{5: true})
	bindNode(si, e.IntNode(2), env)

	// Exact match (name+marks) should win over fallback (name only)
	result, err := lookupNode(si, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(2)) {
		t.Errorf("expected exact match (2), got %s", result.Repr())
	}
}

func TestLookupFallbackDoesNotApplyToUnmarked(t *testing.T) {
	env := NewEnvironment()
	// Don't bind "y" at all
	_, err := lookupNode(e.IdentNode("y"), env)
	if err == nil {
		t.Error("expected error for undefined identifier")
	}
}

func TestResolveCallableHeadWithScopedIdentifier(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		fst := args[0].IntVal
		snd := args[1].IntVal
		return e.IntNode(fst + snd), nil
	})

	// Call (+ 1 2) where + is a ScopedIdentifier — should resolve via fallback
	expr := e.NewListNode([]*e.Node{
		e.ScopedIdentNode("+", map[uint64]bool{1: true}),
		e.IntNode(1),
		e.IntNode(2),
	})

	evaluator := New(engraving.StandardLogger, env)
	result, err := evaluator.EvaluateNode(context.Background(), e.NewListNode([]*e.Node{expr}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(3)) {
		t.Errorf("expected 3, got %s", result.Repr())
	}
}

func TestResolveCallableHeadWithNonIdentifier(t *testing.T) {
	env := NewEnvironment()
	// A list whose head is an integer — not resolvable as a transformer,
	// falls through to normal function call which will fail
	expr := e.NewListNode([]*e.Node{e.NewListNode([]*e.Node{e.IntNode(1)})})

	evaluator := New(engraving.StandardLogger, env)
	_, err := evaluator.EvaluateNode(context.Background(), e.NewListNode([]*e.Node{expr}))
	if err == nil {
		t.Error("expected error when calling a non-procedure")
	}
}

func TestScopedIdentifierLookupDuringEvaluation(t *testing.T) {
	env := NewEnvironment()

	// Bind a ScopedIdentifier
	bindNode(e.ScopedIdentNode("x", map[uint64]bool{1: true}), e.IntNode(42), env)

	// Evaluate an expression tree that references the ScopedIdentifier
	// (define y <scoped-x>) y
	expr := e.NewListNode([]*e.Node{
		e.NewListNode([]*e.Node{e.IdentNode("define"), e.IdentNode("y"), e.ScopedIdentNode("x", map[uint64]bool{1: true})}),
		e.IdentNode("y"),
	})

	evaluator := New(engraving.StandardLogger, env)
	result, err := evaluator.EvaluateNode(context.Background(), expr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestScopedIdentifierAndPlainIdentifierAreDistinct(t *testing.T) {
	env := NewEnvironment()

	// Bind plain "x" = 1
	bindNode(e.IdentNode("x"), e.IntNode(1), env)
	// Bind scoped "x" with mark = 2
	bindNode(e.ScopedIdentNode("x", map[uint64]bool{1: true}), e.IntNode(2), env)

	// Evaluate plain x — should get 1
	expr1 := e.NewListNode([]*e.Node{e.IdentNode("x")})
	evaluator1 := New(engraving.StandardLogger, env)
	result1, err := evaluator1.EvaluateNode(context.Background(), expr1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result1.Equiv(e.IntNode(1)) {
		t.Errorf("plain x should be 1, got %s", result1.Repr())
	}

	// Evaluate scoped x — should get 2
	expr2 := e.NewListNode([]*e.Node{e.ScopedIdentNode("x", map[uint64]bool{1: true})})
	evaluator2 := New(engraving.StandardLogger, env)
	result2, err := evaluator2.EvaluateNode(context.Background(), expr2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result2.Equiv(e.IntNode(2)) {
		t.Errorf("scoped x should be 2, got %s", result2.Repr())
	}
}
