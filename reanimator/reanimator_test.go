package reanimator

import (
	"strings"
	"testing"

	"github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
	"github.com/archevel/ghoul/engraving"
	"github.com/archevel/ghoul/exhumer"
)

func parseNodes(t *testing.T, code string) *bones.Node {
	t.Helper()
	res, parsed := exhumer.Parse(strings.NewReader(code))
	if res != 0 {
		t.Fatalf("failed to parse: %s", code)
	}
	return parsed.Expressions
}

func newTestReanimator() *Reanimator {
	var counter uint64
	return New(engraving.StandardLogger, &counter)
}

// --- Core macro expansion tests ---

func TestExpandSyntaxRulesSimple(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax add-one (syntax-rules () ((add-one x) (+ x 1))))
(add-one 5)
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// define-syntax stripped, macro call expanded: should have one result
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), reprAll(results))
	}

	// The expanded expression should be (+ 5 1), not (add-one 5)
	repr := results[0].Repr()
	if strings.Contains(repr, "add-one") {
		t.Errorf("macro call should be expanded, but got: %s", repr)
	}
	// Should be a CallNode with children [+, 5, 1]
	if results[0].Kind != bones.CallNode {
		t.Errorf("expected CallNode, got kind %d: %s", results[0].Kind, repr)
	}
}

func TestExpandGeneralTransformerSimple(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax always-42 (lambda (stx) 42))
(always-42 anything)
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// The expanded expression should be 42
	if results[0].Kind != bones.IntegerNode || results[0].IntVal != 42 {
		t.Errorf("expected 42, got: %s", results[0].Repr())
	}
}

func TestExpandDefineSyntaxIsStripped(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax my-mac (syntax-rules () ((my-mac) 42)))
(define x 10)
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Should have one expression: (define x 10). define-syntax stripped.
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), reprAll(results))
	}
	if results[0].Kind != bones.DefineNode {
		t.Errorf("expected DefineNode, got kind %d: %s", results[0].Kind, results[0].Repr())
	}
	// DefineNode has children [name, value]
	if len(results[0].Children) != 2 {
		t.Fatalf("expected 2 children in DefineNode, got %d", len(results[0].Children))
	}
	if results[0].Children[0].IdentName() != "x" {
		t.Errorf("expected name 'x', got: %s", results[0].Children[0].Repr())
	}
	if results[0].Children[1].Kind != bones.IntegerNode || results[0].Children[1].IntVal != 10 {
		t.Errorf("expected value 10, got: %s", results[0].Children[1].Repr())
	}
}

func TestExpandMetaMacro(t *testing.T) {
	// A macro that expands to define-syntax should register the new macro,
	// which can then be used.
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax def-adder (syntax-rules ()
  ((def-adder name val)
   (define-syntax name (syntax-rules () ((name x) (+ x val)))))))
(def-adder add-five 5)
(add-five 10)
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Should have one expression: the expansion of (add-five 10) = (+ 10 5)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), reprAll(results))
	}
	repr := results[0].Repr()
	if strings.Contains(repr, "add-five") || strings.Contains(repr, "def-adder") {
		t.Errorf("macro calls should be fully expanded, but got: %s", repr)
	}
}

func TestExpandPreservesNonMacroCode(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define x 42)
(+ x 1)
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Should have two expressions, unchanged
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Kind != bones.DefineNode {
		t.Errorf("expected DefineNode, got kind %d", results[0].Kind)
	}
	if results[1].Kind != bones.CallNode {
		t.Errorf("expected CallNode, got kind %d", results[1].Kind)
	}
}

func TestExpandHygienePreserved(t *testing.T) {
	// The classic hygiene test: macro introduces "tmp", user also has "tmp".
	// After expansion, the macro's tmp should have hygiene marks.
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax my-or (syntax-rules ()
  ((my-or a b) (begin (define tmp a) (cond (tmp tmp) (else b))))))
(define tmp 5)
(my-or #f tmp)
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Verify the expansion happened (my-or should not appear)
	allRepr := reprAll(results)
	if strings.Contains(allRepr, "my-or") {
		t.Errorf("macro call should be expanded, but got: %s", allRepr)
	}
}

// --- Tests for macro calls nested inside core forms ---

func TestExpandMacroInsideDefine(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(define y (wrap 5))
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// (define y (+ 5 1))
	repr := results[0].Repr()
	if strings.Contains(repr, "wrap") {
		t.Errorf("macro inside define should be expanded, got: %s", repr)
	}
	if results[0].Kind != bones.DefineNode {
		t.Errorf("expected DefineNode, got kind %d", results[0].Kind)
	}
}

func TestExpandMacroInsideCond(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax tt (syntax-rules () ((tt) #t)))
(cond ((tt) 42))
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	repr := results[0].Repr()
	if strings.Contains(repr, "tt") {
		t.Errorf("macro inside cond should be expanded, got: %s", repr)
	}
}

func TestExpandMacroInsideBegin(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(begin (wrap 1) (wrap 2))
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	repr := results[0].Repr()
	if strings.Contains(repr, "wrap") {
		t.Errorf("macro inside begin should be expanded, got: %s", repr)
	}
	if results[0].Kind != bones.BeginNode {
		t.Errorf("expected BeginNode, got kind %d", results[0].Kind)
	}
}

func TestExpandMacroInsideLambdaBody(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(lambda (x) (wrap x))
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	repr := results[0].Repr()
	if strings.Contains(repr, "wrap") {
		t.Errorf("macro inside lambda body should be expanded, got: %s", repr)
	}
	if results[0].Kind != bones.LambdaNode {
		t.Errorf("expected LambdaNode, got kind %d", results[0].Kind)
	}
}

func TestExpandMacroInsideFunctionCallArgs(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(foo (wrap 1) (wrap 2))
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	repr := results[0].Repr()
	if strings.Contains(repr, "wrap") {
		t.Errorf("macro inside function call args should be expanded, got: %s", repr)
	}
	if results[0].Kind != bones.CallNode {
		t.Errorf("expected CallNode, got kind %d", results[0].Kind)
	}
}

// --- Inner define-syntax scoping ---

func TestExpandDefineSyntaxInsideLambda(t *testing.T) {
	// define-syntax inside a lambda body creates a locally-scoped macro
	r := newTestReanimator()
	nodes := parseNodes(t, `
(lambda (x)
  (define-syntax local-mac (syntax-rules () ((local-mac) 99)))
  (local-mac))
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	repr := results[0].Repr()
	if strings.Contains(repr, "define-syntax") {
		t.Errorf("inner define-syntax should be stripped, got: %s", repr)
	}
	if strings.Contains(repr, "local-mac") {
		t.Errorf("inner macro call should be expanded, got: %s", repr)
	}
	if results[0].Kind != bones.LambdaNode {
		t.Errorf("expected LambdaNode, got kind %d", results[0].Kind)
	}
}

func TestExpandDefineSyntaxInsideBegin(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(begin
  (define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
  (wrap 5))
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	repr := results[0].Repr()
	if strings.Contains(repr, "define-syntax") || strings.Contains(repr, "wrap") {
		t.Errorf("define-syntax should be stripped and macro expanded, got: %s", repr)
	}
	if results[0].Kind != bones.BeginNode {
		t.Errorf("expected BeginNode, got kind %d", results[0].Kind)
	}
}

// --- Error path tests ---

func TestExpandDefineSyntaxMissingName(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(define-syntax)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestExpandDefineSyntaxNonIdentifierName(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(define-syntax 42 (syntax-rules () ((foo) 1)))`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-identifier name")
	}
}

func TestExpandDefineSyntaxMissingTransformer(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(define-syntax foo)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for missing transformer")
	}
}

func TestExpandDefineSyntaxNonListTransformer(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(define-syntax foo 42)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-list transformer")
	}
}

func TestExpandDefineSyntaxBadSyntaxRules(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(define-syntax foo (syntax-rules))`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for malformed syntax-rules")
	}
}

func TestExpandDefineSyntaxNonProcedureTransformer(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(define-syntax foo (begin 42))`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-procedure transformer")
	}
}

// --- Edge cases ---

func TestExpandEmptyInput(t *testing.T) {
	r := newTestReanimator()
	results, err := r.ReanimateNodes(nil)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil, got: %v", reprAll(results))
	}
}

func TestExpandEmptyInputNilNode(t *testing.T) {
	r := newTestReanimator()
	results, err := r.ReanimateNodes(bones.Nil)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil, got: %v", reprAll(results))
	}
}

func TestExpandQuoteNotRecursed(t *testing.T) {
	// Macro calls inside quote should NOT be expanded
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(quote (wrap 5))
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// quote should preserve macro call as-is
	repr := results[0].Repr()
	if !strings.Contains(repr, "wrap") {
		t.Errorf("quote should preserve macro call as-is, got: %s", repr)
	}
	if results[0].Kind != bones.QuoteNode {
		t.Errorf("expected QuoteNode, got kind %d", results[0].Kind)
	}
}

func TestExpandMacroCallError(t *testing.T) {
	// A syntax-rules macro with no matching pattern should error
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax only-one-arg (syntax-rules () ((only-one-arg x) x)))
(only-one-arg 1 2 3)
`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-matching macro call")
	}
}

func TestExpandMacroLocationSetOnExpansion(t *testing.T) {
	// Expanded code should have macro call site location set
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(wrap 5)
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// The result should be a translated CallNode for (+ 5 1).
	// Its Loc should be set from the macro call site.
	result := results[0]
	if result.Kind != bones.CallNode {
		t.Fatalf("expected CallNode, got kind %d: %s", result.Kind, result.Repr())
	}
	if result.Loc == nil {
		t.Error("expected location to be set on expanded code")
	}
}

// --- Integration test: expand then evaluate ---

func TestExpandIntegrationWithEvaluator(t *testing.T) {
	var counter uint64
	r := New(engraving.StandardLogger, &counter)
	nodes := parseNodes(t, `
(define-syntax add-one (syntax-rules () ((add-one x) (+ x 1))))
(add-one 41)
`)
	results, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}

	// Evaluate the expanded code
	env := ev.NewEnvironment()
	env.Register("+", func(args []*bones.Node, evaluator *ev.Evaluator) (*bones.Node, error) {
		fst := args[0].IntVal
		snd := args[1].IntVal
		return bones.IntNode(fst + snd), nil
	})
	evaluator := ev.NewWithMarkCounter(engraving.StandardLogger, env, &counter)
	result, err := evaluator.ConsumeNodes(results)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	if result.Kind != bones.IntegerNode || result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

// --- Helpers ---

func reprAll(nodes []*bones.Node) string {
	var parts []string
	for _, n := range nodes {
		parts = append(parts, n.Repr())
	}
	return strings.Join(parts, " ")
}
