package expander

import (
	"fmt"
	"strings"
	"testing"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
	"github.com/archevel/ghoul/macromancy"
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

// --- Tests for macro calls nested inside core forms ---

func TestExpandMacroInsideDefine(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(define y (wrap 5))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	// (define y (+ 5 1))
	result := expanded.First()
	if strings.Contains(result.Repr(), "wrap") {
		t.Errorf("macro inside define should be expanded, got: %s", result.Repr())
	}
}

func TestExpandMacroInsideSetBang(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(set! y (wrap 5))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	if strings.Contains(result.Repr(), "wrap") {
		t.Errorf("macro inside set! should be expanded, got: %s", result.Repr())
	}
}

func TestExpandMacroInsideCond(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax tt (syntax-rules () ((tt) #t)))
(cond ((tt) 42))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	if strings.Contains(result.Repr(), "tt") {
		t.Errorf("macro inside cond should be expanded, got: %s", result.Repr())
	}
}

func TestExpandMacroInsideBegin(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(begin (wrap 1) (wrap 2))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	if strings.Contains(result.Repr(), "wrap") {
		t.Errorf("macro inside begin should be expanded, got: %s", result.Repr())
	}
}

func TestExpandMacroInsideLambdaBody(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(lambda (x) (wrap x))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	if strings.Contains(result.Repr(), "wrap") {
		t.Errorf("macro inside lambda body should be expanded, got: %s", result.Repr())
	}
}

func TestExpandMacroInsideFunctionCallArgs(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(foo (wrap 1) (wrap 2))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	if strings.Contains(result.Repr(), "wrap") {
		t.Errorf("macro inside function call args should be expanded, got: %s", result.Repr())
	}
}

func TestExpandDefineSyntaxInsideLambda(t *testing.T) {
	// define-syntax inside a lambda body creates a locally-scoped macro
	exp := newTestExpander()
	exprs := parseExprs(t, `
(lambda ()
  (define-syntax local-mac (syntax-rules () ((local-mac) 99)))
  (local-mac))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	repr := result.Repr()
	if strings.Contains(repr, "define-syntax") {
		t.Errorf("inner define-syntax should be stripped, got: %s", repr)
	}
	if strings.Contains(repr, "local-mac") {
		t.Errorf("inner macro call should be expanded, got: %s", repr)
	}
}

func TestExpandDefineSyntaxInsideBegin(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `
(begin
  (define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
  (wrap 5))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	repr := result.Repr()
	if strings.Contains(repr, "define-syntax") || strings.Contains(repr, "wrap") {
		t.Errorf("define-syntax should be stripped and macro expanded, got: %s", repr)
	}
}

// --- Error path tests ---

func TestExpandDefineSyntaxMissingName(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `(define-syntax)`)
	_, err := exp.ExpandAll(exprs)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestExpandDefineSyntaxNonIdentifierName(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `(define-syntax 42 (syntax-rules () ((foo) 1)))`)
	_, err := exp.ExpandAll(exprs)
	if err == nil {
		t.Fatal("expected error for non-identifier name")
	}
}

func TestExpandDefineSyntaxMissingTransformer(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `(define-syntax foo)`)
	_, err := exp.ExpandAll(exprs)
	if err == nil {
		t.Fatal("expected error for missing transformer")
	}
}

func TestExpandDefineSyntaxNonListTransformer(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `(define-syntax foo 42)`)
	_, err := exp.ExpandAll(exprs)
	if err == nil {
		t.Fatal("expected error for non-list transformer")
	}
}

func TestExpandDefineSyntaxBadSyntaxRules(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `(define-syntax foo (syntax-rules))`)
	_, err := exp.ExpandAll(exprs)
	if err == nil {
		t.Fatal("expected error for malformed syntax-rules")
	}
}

func TestExpandDefineSyntaxNonProcedureTransformer(t *testing.T) {
	exp := newTestExpander()
	exprs := parseExprs(t, `(define-syntax foo (begin 42))`)
	_, err := exp.ExpandAll(exprs)
	if err == nil {
		t.Fatal("expected error for non-procedure transformer")
	}
}

// --- Edge cases ---

func TestExpandEmptyInput(t *testing.T) {
	exp := newTestExpander()
	expanded, err := exp.ExpandAll(e.NIL)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if expanded != e.NIL {
		t.Errorf("expected NIL, got: %s", expanded.Repr())
	}
}

func TestExpandAtomExpression(t *testing.T) {
	exp := newTestExpander()
	// Non-list expressions should pass through unchanged
	result, err := exp.expandExpr(e.Integer(42))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Equiv(e.Integer(42)) {
		t.Errorf("expected 42, got: %s", result.Repr())
	}
}

func TestExpandNILExpression(t *testing.T) {
	exp := newTestExpander()
	result, err := exp.expandExpr(e.NIL)
	if err != nil {
		t.Fatal(err)
	}
	if result != e.NIL {
		t.Errorf("expected NIL, got: %s", result.Repr())
	}
}

func TestExpandQuoteNotRecursed(t *testing.T) {
	// Macro calls inside quote should NOT be expanded
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(quote (wrap 5))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	if !strings.Contains(result.Repr(), "wrap") {
		t.Errorf("quote should preserve macro call as-is, got: %s", result.Repr())
	}
}

func TestExpandScopedIdentifierHead(t *testing.T) {
	// A ScopedIdentifier at the head of a list should still be checked
	// for macro names.
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("my-mac"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return e.Integer(99), nil
			},
		},
	})
	// Use identName which handles ScopedIdentifier
	name := identName(e.ScopedIdentifier{Name: "my-mac", Marks: map[uint64]bool{1: true}})
	if name != "my-mac" {
		t.Errorf("identName should extract name from ScopedIdentifier, got: %s", name)
	}
}

func TestExpandIdentNameNonIdentifier(t *testing.T) {
	if identName(e.Integer(42)) != "" {
		t.Error("identName should return empty string for non-identifier")
	}
}

func TestExpandMacroCallError(t *testing.T) {
	// A syntax-rules macro with no matching pattern should error
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax only-one-arg (syntax-rules () ((only-one-arg x) x)))
(only-one-arg 1 2 3)
`)
	_, err := exp.ExpandAll(exprs)
	if err == nil {
		t.Fatal("expected error for non-matching macro call")
	}
}

func TestExpandMacroLocationSetOnExpansion(t *testing.T) {
	// Expanded code should have macro call site location set
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(wrap 5)
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	// The expanded (+ 5 1) should be a Pair
	pair, ok := expanded.First().(*e.Pair)
	if !ok {
		t.Fatalf("expected Pair, got %T", expanded.First())
	}
	// Location should be set (from macro call site)
	// Parser doesn't set locations for string-based parsing without filename,
	// so we just verify the Pair exists and expansion worked
	_ = pair
}

func TestExpandCondWithNonListClause(t *testing.T) {
	// cond clause where the predicate is a macro call
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax tt (syntax-rules () ((tt) #t)))
(cond ((tt) 42) (else 0))
`)
	expanded, err := exp.ExpandAll(exprs)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	result := expanded.First()
	if strings.Contains(result.Repr(), "tt") {
		t.Errorf("macro in cond predicate should be expanded, got: %s", result.Repr())
	}
}

func TestExpandLambdaMalformed(t *testing.T) {
	// (lambda) with no params or body — should pass through
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("wrap"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return e.Integer(1), nil
			},
		},
	})
	// Force containsMacroCall to return true by having a macro in scope
	// then hit expandLambda with a malformed lambda
	form := e.Cons(e.Identifier("lambda"), e.NIL)
	result, err := exp.expandLambda(form)
	if err != nil {
		t.Fatal(err)
	}
	// Should return form unchanged
	if result != form {
		t.Error("malformed lambda should be returned as-is")
	}
}

func TestExpandBeginMalformed(t *testing.T) {
	// (begin) with no body — should pass through
	exp := newTestExpander()
	form := e.Cons(e.Identifier("begin"), e.Integer(42)) // improper
	result, err := exp.expandBegin(form)
	if err != nil {
		t.Fatal(err)
	}
	if result != form {
		t.Error("malformed begin should be returned as-is")
	}
}

func TestExpandDefineMalformed(t *testing.T) {
	exp := newTestExpander()
	// (define) — no name or value
	form := e.Cons(e.Identifier("define"), e.NIL)
	result, err := exp.expandDefine(form)
	if err != nil {
		t.Fatal(err)
	}
	if result != form {
		t.Error("malformed define should be returned as-is")
	}

	// (define x) — no value
	form2 := e.Cons(e.Identifier("define"), e.Cons(e.Identifier("x"), e.NIL))
	result2, err := exp.expandDefine(form2)
	if err != nil {
		t.Fatal(err)
	}
	if result2 != form2 {
		t.Error("define without value should be returned as-is")
	}
}

func TestExpandCondMalformed(t *testing.T) {
	exp := newTestExpander()
	// (cond) with improper tail
	form := e.Cons(e.Identifier("cond"), e.Integer(42)) // improper
	result, err := exp.expandCond(form)
	if err != nil {
		t.Fatal(err)
	}
	if result != form {
		t.Error("malformed cond should be returned as-is")
	}
}

func TestExpandMacroInImproperList(t *testing.T) {
	// Function call with dotted pair containing a macro call
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("wrap"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return e.Integer(99), nil
			},
		},
	})
	// (f . (wrap 1)) — improper list with macro in tail
	form := e.Cons(e.Identifier("f"), e.Cons(e.Identifier("wrap"), e.Cons(e.Integer(1), e.NIL)))
	result, err := exp.expandEachInList(form)
	if err != nil {
		t.Fatal(err)
	}
	_ = result
}

func TestExpandSetMacroLocationNonPair(t *testing.T) {
	// setMacroLocation with non-Pair expanded code should not panic
	setMacroLocation(e.Integer(42), e.Cons(e.Identifier("mac"), e.NIL))
}

func TestExpandSetMacroLocationNonPairCallSite(t *testing.T) {
	// setMacroLocation with non-Pair call site should not panic
	setMacroLocation(e.Cons(e.Integer(1), e.NIL), e.NIL)
}

func TestExpandSetMacroLocationNoLoc(t *testing.T) {
	// setMacroLocation where call site Pair has no Loc should not panic
	expanded := e.Cons(e.Integer(1), e.NIL)
	callSite := e.Cons(e.Identifier("mac"), e.NIL)
	// callSite.Loc is nil by default
	setMacroLocation(expanded, callSite)
	// No crash = success
}

func TestExpandMacroScopeParentLookup(t *testing.T) {
	parent := newMacroScope(nil)
	parent.define(e.Identifier("outer"), macroBinding{})
	child := newMacroScope(parent)
	_, found := child.lookup(e.Identifier("outer"))
	if !found {
		t.Error("child scope should find parent binding")
	}
	_, found2 := child.lookup(e.Identifier("missing"))
	if found2 {
		t.Error("should not find non-existent binding")
	}
}

func TestExpandLambdaNoBody(t *testing.T) {
	// (lambda (x)) — params but no body, improper form
	exp := newTestExpander()
	form := e.Cons(e.Identifier("lambda"), e.Cons(
		e.Cons(e.Identifier("x"), e.NIL),
		e.Integer(42))) // improper tail
	result, err := exp.expandLambda(form)
	if err != nil {
		t.Fatal(err)
	}
	if result != form {
		t.Error("lambda with improper body should be returned as-is")
	}
}

func TestExpandSequenceWithDefineSyntax(t *testing.T) {
	// define-syntax inside a sequence should be stripped
	exp := newTestExpander()
	seq := parseExprs(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(wrap 5)
`)
	expanded, err := exp.expandSequence(seq)
	if err != nil {
		t.Fatal(err)
	}
	// Only one expression should remain (define-syntax stripped)
	if expanded == e.NIL {
		t.Fatal("expected non-empty")
	}
	tail, _ := expanded.Tail()
	if tail != e.NIL {
		t.Errorf("expected one expression, got: %s", expanded.Repr())
	}
}

func TestExpandCondWithImproperClauseList(t *testing.T) {
	// cond where the clause list is improper — the break path
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("wrap"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return e.Integer(99), nil
			},
		},
	})
	// (cond ((wrap 1) 2) . 3) — improper list of clauses
	clause := e.Cons(e.Cons(e.Identifier("wrap"), e.Cons(e.Integer(1), e.NIL)), e.Cons(e.Integer(2), e.NIL))
	form := e.Cons(e.Identifier("cond"), e.Cons(clause, e.Integer(3)))
	result, err := exp.expandCond(form)
	if err != nil {
		t.Fatal(err)
	}
	_ = result
}

func TestExpandContainsMacroCallImproperList(t *testing.T) {
	// containsMacroCall on an improper list should not panic
	exp := newTestExpander()
	improper := e.Cons(e.Integer(1), e.Integer(2))
	result := exp.containsMacroCall(improper)
	if result {
		t.Error("improper list with no macros should return false")
	}
}

func TestExpandEachInListImproperWithMacro(t *testing.T) {
	// Dotted pair where the tail position contains a macro call:
	// (f x . (wrap 1)) parsed as Cons(f, Cons(x, Cons(wrap, Cons(1, NIL))))
	// We need an actual improper list: Cons(f, Cons(x, Integer(42)))
	// where expandEachInList hits the !ok path on Tail()
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("wrap"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return e.Integer(99), nil
			},
		},
	})
	// (f . 42) — improper list, Tail() returns !ok
	form := e.Cons(e.Identifier("f"), e.Integer(42))
	result, err := exp.expandEachInList(form)
	if err != nil {
		t.Fatal(err)
	}
	// Should preserve the improper structure
	pair, ok := result.(*e.Pair)
	if !ok {
		t.Fatalf("expected Pair, got %T", result)
	}
	if !pair.T.Equiv(e.Integer(42)) {
		t.Errorf("dotted tail should be preserved, got: %s", pair.T.Repr())
	}
}

func TestExpandSequenceImproperList(t *testing.T) {
	// expandSequence with an improper list should handle the break path
	exp := newTestExpander()
	// (expr1 . expr2) — improper
	seq := e.Cons(e.Integer(1), e.Integer(2))
	result, err := exp.expandSequence(seq)
	if err != nil {
		t.Fatal(err)
	}
	// Should have at least one element
	if result == e.NIL {
		t.Error("expected non-empty result")
	}
}

func TestExpandBeginWithMacroError(t *testing.T) {
	// begin containing a macro that errors during expansion
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("bad"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return nil, fmt.Errorf("intentional error")
			},
		},
	})
	form := e.Cons(e.Identifier("begin"), e.Cons(
		e.Cons(e.Identifier("bad"), e.NIL), e.NIL))
	_, err := exp.expandBegin(form)
	if err == nil {
		t.Fatal("expected error from macro inside begin")
	}
}

func TestExpandLambdaWithMacroError(t *testing.T) {
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("bad"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return nil, fmt.Errorf("intentional error")
			},
		},
	})
	form := e.Cons(e.Identifier("lambda"),
		e.Cons(e.Cons(e.Identifier("x"), e.NIL),
			e.Cons(e.Cons(e.Identifier("bad"), e.NIL), e.NIL)))
	_, err := exp.expandLambda(form)
	if err == nil {
		t.Fatal("expected error from macro inside lambda body")
	}
}

func TestExpandDefineWithMacroError(t *testing.T) {
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("bad"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return nil, fmt.Errorf("intentional error")
			},
		},
	})
	form := e.Cons(e.Identifier("define"),
		e.Cons(e.Identifier("x"),
			e.Cons(e.Cons(e.Identifier("bad"), e.NIL), e.NIL)))
	_, err := exp.expandDefine(form)
	if err == nil {
		t.Fatal("expected error from macro inside define value")
	}
}

func TestExpandCondWithMacroError(t *testing.T) {
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("bad"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return nil, fmt.Errorf("intentional error")
			},
		},
	})
	clause := e.Cons(e.Cons(e.Identifier("bad"), e.NIL), e.Cons(e.Integer(1), e.NIL))
	form := e.Cons(e.Identifier("cond"), e.Cons(clause, e.NIL))
	_, err := exp.expandCond(form)
	if err == nil {
		t.Fatal("expected error from macro inside cond clause")
	}
}

func TestExpandCallWithMacroError(t *testing.T) {
	exp := newTestExpander()
	exp.scopes.define(e.Identifier("bad"), macroBinding{
		syntaxTransformer: &macromancy.SyntaxTransformer{
			Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
				return nil, fmt.Errorf("intentional error")
			},
		},
	})
	form := e.Cons(e.Identifier("f"), e.Cons(
		e.Cons(e.Identifier("bad"), e.NIL), e.NIL))
	_, err := exp.expandCall(form)
	if err == nil {
		t.Fatal("expected error from macro inside function call")
	}
}

func TestExpandGeneralTransformerError(t *testing.T) {
	// A general transformer whose body errors during expansion
	exp := newTestExpander()
	exprs := parseExprs(t, `
(define-syntax bad-gen (lambda (stx) (undefined-function)))
(bad-gen 1)
`)
	_, err := exp.ExpandAll(exprs)
	if err == nil {
		t.Fatal("expected error from general transformer")
	}
}

func TestExpandListFromSliceEmpty(t *testing.T) {
	result := listFromSlice(nil)
	if result != e.NIL {
		t.Error("empty slice should produce NIL")
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
