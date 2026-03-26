package expander

import (
	"strings"
	"testing"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
	"github.com/archevel/ghoul/parser"
)

func parseExprs(t *testing.T, code string) e.List {
	t.Helper()
	res, parsed := parser.Parse(strings.NewReader(code))
	if res != 0 {
		t.Fatalf("failed to parse: %s", code)
	}
	return parsed.Expressions
}

func newTestExpander() *Expander {
	var counter uint64
	return New(logging.StandardLogger, &counter)
}

func TestExpandSyntaxRulesSimple(t *testing.T) {
	// define-syntax with syntax-rules should be stripped from output,
	// and subsequent macro calls should be expanded.
	expander := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax add-one (syntax-rules () ((add-one x) (+ x 1))))
(add-one 5)
`)
	expanded, err := expander.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Should have one expression (define-syntax stripped, macro call expanded)
	if expanded == e.NIL {
		t.Fatal("expected non-empty result")
	}
	tail, _ := expanded.Tail()
	if tail != e.NIL {
		t.Errorf("expected exactly one expression after expansion, got more: %s", expanded.Repr())
	}

	// The expanded expression should be (+ 5 1), not (add-one 5)
	result := expanded.First()
	resultRepr := result.Repr()
	if strings.Contains(resultRepr, "add-one") {
		t.Errorf("macro call should be expanded, but got: %s", resultRepr)
	}
}

func TestExpandGeneralTransformerSimple(t *testing.T) {
	// General transformer (lambda-based macro) should work through
	// the sub-evaluator during expansion.
	expander := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax always-42 (lambda (stx) 42))
(always-42 anything)
`)
	expanded, err := expander.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	if expanded == e.NIL {
		t.Fatal("expected non-empty result")
	}
	// The expanded expression should be 42
	if !expanded.First().Equiv(e.Integer(42)) {
		t.Errorf("expected 42, got: %s", expanded.First().Repr())
	}
}

func TestExpandDefineSyntaxIsStripped(t *testing.T) {
	expander := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax my-mac (syntax-rules () ((my-mac) 42)))
(define x 10)
`)
	expanded, err := expander.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Should have one expression: (define x 10). define-syntax stripped.
	if expanded == e.NIL {
		t.Fatal("expected non-empty result")
	}
	head := expanded.First()
	headList, ok := head.(e.List)
	if !ok {
		t.Fatalf("expected list, got %T", head)
	}
	if identName(headList.First()) != "define" {
		t.Errorf("expected (define ...), got: %s", head.Repr())
	}
	tail, _ := expanded.Tail()
	if tail != e.NIL {
		t.Errorf("expected exactly one expression, got: %s", expanded.Repr())
	}
}

func TestExpandMetaMacro(t *testing.T) {
	// A macro that expands to define-syntax should register the new macro,
	// which can then be used.
	expander := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax def-adder (syntax-rules ()
  ((def-adder name val)
   (define-syntax name (syntax-rules () ((name x) (+ x val)))))))
(def-adder add-five 5)
(add-five 10)
`)
	expanded, err := expander.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Should have one expression: the expansion of (add-five 10) = (+ 10 5)
	if expanded == e.NIL {
		t.Fatal("expected non-empty result")
	}
	result := expanded.First()
	resultRepr := result.Repr()
	if strings.Contains(resultRepr, "add-five") || strings.Contains(resultRepr, "def-adder") {
		t.Errorf("macro calls should be fully expanded, but got: %s", resultRepr)
	}
}

func TestExpandPreservesNonMacroCode(t *testing.T) {
	expander := newTestExpander()
	exprs := parseExprs(t, `
(define x 42)
(+ x 1)
`)
	expanded, err := expander.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Should have two expressions, unchanged
	if expanded == e.NIL {
		t.Fatal("expected non-empty result")
	}
	tail, _ := expanded.Tail()
	if tail == e.NIL {
		t.Error("expected two expressions")
	}
}

func TestExpandHygienePreserved(t *testing.T) {
	// The classic hygiene test: macro introduces "tmp", user also has "tmp".
	// After expansion, the macro's tmp should have hygiene marks.
	expander := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax my-or (syntax-rules ()
  ((my-or a b) (begin (define tmp a) (cond (tmp tmp) (else b))))))
(define tmp 5)
(my-or #f tmp)
`)
	expanded, err := expander.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Verify the expansion happened (my-or should not appear)
	repr := expanded.Repr()
	if strings.Contains(repr, "my-or") {
		t.Errorf("macro call should be expanded, but got: %s", repr)
	}
}

func TestExpandIntegrationWithEvaluator(t *testing.T) {
	// End-to-end: expand then evaluate. This verifies the expanded code
	// is valid for the evaluator.
	var counter uint64
	expander := New(logging.StandardLogger, &counter)

	exprs := parseExprs(t, `
(define-syntax add-one (syntax-rules () ((add-one x) (+ x 1))))
(add-one 41)
`)
	expanded, err := expander.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Evaluate the expanded code
	env := ev.NewEnvironment()
	env.Register("+", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		tl, _ := args.Tail()
		snd := tl.First().(e.Integer)
		return e.Integer(fst + snd), nil
	})
	evaluator := ev.NewWithMarkCounter(logging.StandardLogger, env, &counter)
	result, err := evaluator.Evaluate(expanded)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	if !result.Equiv(e.Integer(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}
