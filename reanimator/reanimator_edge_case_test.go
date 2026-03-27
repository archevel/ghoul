package reanimator

import (
	"strings"
	"testing"

	"github.com/archevel/ghoul/bones"
)

// --- Expansion error paths ---

func TestExpandNodeDefineWithMacroErrorInValue(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax fail-mac (syntax-rules () ((fail-mac x y) x)))
(define x (fail-mac 1))
`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-matching macro in define value")
	}
}

func TestExpandNodeSetBangWithMacroError(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax fail-mac (syntax-rules () ((fail-mac x y) x)))
(define x 1)
(set! x (fail-mac 1))
`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-matching macro in set! value")
	}
}

func TestExpandNodeCondWithMacroErrorInClause(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax fail-mac (syntax-rules () ((fail-mac x y) x)))
(cond ((fail-mac 1) 42))
`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-matching macro in cond clause")
	}
}

func TestExpandNodeCondWithAtomClause(t *testing.T) {
	r := newTestReanimator()
	// Define a macro, then use it so containsMacroCall triggers
	// but have a non-list clause
	nodes := parseNodes(t, `
(define-syntax id (syntax-rules () ((id x) x)))
(cond (id 42))
`)
	// The cond clause "id" is not a list — should pass through as atom
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected result")
	}
}

func TestExpandNodeBeginWithMacroError(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax fail-mac (syntax-rules () ((fail-mac x y) x)))
(begin (fail-mac 1))
`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-matching macro in begin body")
	}
}

func TestExpandNodeLambdaWithMacroError(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax fail-mac (syntax-rules () ((fail-mac x y) x)))
(lambda (x) (fail-mac 1))
`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-matching macro in lambda body")
	}
}

func TestExpandNodeCallWithMacroError(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax fail-mac (syntax-rules () ((fail-mac x y) x)))
(+ (fail-mac 1) 2)
`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-matching macro in call args")
	}
}

// --- expandEach with non-list ---

func TestExpandNodeEachNonList(t *testing.T) {
	r := newTestReanimator()
	scope := newMacroScope(nil)
	result, err := r.expandEach(bones.IntNode(42), scope)
	if err != nil {
		t.Fatal(err)
	}
	if result.IntVal != 42 {
		t.Errorf("expected 42 passthrough, got %s", result.Repr())
	}
}

// --- setMacroLocation edge cases ---

func TestSetMacroLocationNilExpanded(t *testing.T) {
	// Should not panic
	setMacroLocation(nil, bones.NewListNode([]*bones.Node{bones.IdentNode("mac")}))
}

func TestSetMacroLocationNilCallSite(t *testing.T) {
	setMacroLocation(bones.IntNode(1), nil)
}

func TestSetMacroLocationNoLoc(t *testing.T) {
	callSite := bones.NewListNode([]*bones.Node{bones.IdentNode("mac")})
	// callSite has no Loc — should not set location
	expanded := bones.NewListNode([]*bones.Node{bones.IntNode(1)})
	setMacroLocation(expanded, callSite)
	if expanded.Loc != nil {
		t.Error("should not set location when callSite has no Loc")
	}
}

func TestSetMacroLocationNoChildren(t *testing.T) {
	callSite := &bones.Node{Kind: bones.ListNode, Loc: &bones.SourcePosition{Ln: 1, Col: 1}}
	expanded := bones.NewListNode([]*bones.Node{bones.IntNode(1)})
	setMacroLocation(expanded, callSite)
	// macroName is "" since callSite has no children
	if expanded.Loc == nil {
		t.Error("should set location even with no macro name")
	}
}

// --- Translation error paths ---

func TestTranslateNodeDefineOrSetShort(t *testing.T) {
	// (define x) — missing value
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("define"), bones.IdentNode("x")})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for short define")
	}
}

func TestTranslateNodeSetShortForm(t *testing.T) {
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("set!"), bones.IdentNode("x")})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for short set!")
	}
}

func TestTranslateNodeLambdaNoBodyForm(t *testing.T) {
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("lambda"), bones.NewListNode([]*bones.Node{bones.IdentNode("x")})})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for lambda with no body")
	}
}

func TestTranslateNodeLambdaInvalidParams(t *testing.T) {
	// (lambda 42 body) — 42 is not a valid param list
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("lambda"), bones.IntNode(42), bones.IntNode(1)})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for invalid lambda params")
	}
}

func TestTranslateNodeCondNonListClauseError(t *testing.T) {
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("cond"), bones.IntNode(42)})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for non-list cond clause")
	}
}

func TestTranslateNodeCondEmptyClauseError(t *testing.T) {
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("cond"), bones.NewListNode(nil)})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for empty cond clause")
	}
}

func TestTranslateNodeCondElseClause(t *testing.T) {
	// (cond (else 42)) — else clause
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("cond"),
		bones.NewListNode([]*bones.Node{bones.IdentNode("else"), bones.IntNode(42)}),
	})
	result, err := translateNode(node)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != bones.CondNode || len(result.Clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(result.Clauses))
	}
	if !result.Clauses[0].IsElse {
		t.Error("expected else clause")
	}
}

func TestTranslateNodeCallWithNestedTranslation(t *testing.T) {
	// (f (+ 1 2)) — nested call should be translated
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("f"),
		bones.NewListNode([]*bones.Node{bones.IdentNode("+"), bones.IntNode(1), bones.IntNode(2)}),
	})
	result, err := translateNode(node)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != bones.CallNode {
		t.Fatalf("expected CallNode, got %d", result.Kind)
	}
	if result.Children[1].Kind != bones.CallNode {
		t.Error("nested call should also be translated to CallNode")
	}
}

func TestTranslateNodeBeginWithNestedError(t *testing.T) {
	// (begin (quote)) — quote without arg should error during translation
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("begin"),
		bones.NewListNode([]*bones.Node{bones.IdentNode("quote")}),
	})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for malformed quote inside begin")
	}
}

func TestTranslateNodeLambdaWithNestedError(t *testing.T) {
	// (lambda (x) (quote)) — malformed body
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("lambda"),
		bones.NewListNode([]*bones.Node{bones.IdentNode("x")}),
		bones.NewListNode([]*bones.Node{bones.IdentNode("quote")}),
	})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for malformed quote inside lambda body")
	}
}

func TestTranslateNodeDefineWithNestedError(t *testing.T) {
	// (define x (quote)) — malformed value
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("define"),
		bones.IdentNode("x"),
		bones.NewListNode([]*bones.Node{bones.IdentNode("quote")}),
	})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for malformed quote in define value")
	}
}

func TestTranslateNodeCallWithNestedError(t *testing.T) {
	// (f (quote)) — malformed arg
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("f"),
		bones.NewListNode([]*bones.Node{bones.IdentNode("quote")}),
	})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for malformed quote in call arg")
	}
}

func TestTranslateNodeCondClauseWithNestedError(t *testing.T) {
	// (cond (#t (quote))) — malformed consequent
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("cond"),
		bones.NewListNode([]*bones.Node{
			bones.BoolNode(true),
			bones.NewListNode([]*bones.Node{bones.IdentNode("quote")}),
		}),
	})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for malformed quote in cond consequent")
	}
}

// --- inheritLoc ---

func TestInheritNodeLocChildAlreadyHasLoc(t *testing.T) {
	parent := &bones.Node{Loc: &bones.SourcePosition{Ln: 1, Col: 1}}
	childLoc := &bones.SourcePosition{Ln: 5, Col: 5}
	child := &bones.Node{Loc: childLoc}
	inheritLoc(child, parent)
	// Should keep child's own loc
	if child.Loc != childLoc {
		t.Error("child's own loc should not be overwritten")
	}
}

func TestInheritNodeLocParentNoLoc(t *testing.T) {
	parent := &bones.Node{}
	child := &bones.Node{}
	inheritLoc(child, parent)
	if child.Loc != nil {
		t.Error("should not set loc from parent with no loc")
	}
}

// --- General transformer error ---

func TestExpandGeneralTransformerEvalError(t *testing.T) {
	r := newTestReanimator()
	// A transformer body that references an undefined identifier
	nodes := parseNodes(t, `
(define-syntax bad-transformer (lambda (stx) undefined-var))
(bad-transformer 1)
`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error from general transformer evaluation")
	}
}

func TestExpandGeneralTransformerNonProcedure(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(define-syntax bad (begin 42))`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-procedure transformer")
	}
}

// --- containsMacroCall with deep nesting ---

func TestContainsMacroCallInNestedLambda(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax id (syntax-rules () ((id x) x)))
(lambda (x) (lambda (y) (id y)))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatal(err)
	}
	// Should expand the inner (id y) to just y
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	repr := result[0].Repr()
	if strings.Contains(repr, "id") {
		t.Errorf("macro should be expanded in nested lambda, got: %s", repr)
	}
}

// --- Sequence with define-syntax stripped ---

func TestExpandSequenceStripsDefineSyntax(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(begin
  (define-syntax id (syntax-rules () ((id x) x)))
  (id 42))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result (begin with inner expanded), got %d", len(result))
	}
}
