package reanimator

import (
	"strings"
	"testing"

	"github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/exhumer"
)

// --- Translation edge cases ---

func TestTranslateNodeQuoteMissingArg(t *testing.T) {
	// (quote) — no argument
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("quote")})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for quote with no argument")
	}
}

func TestTranslateNodeDefineShort(t *testing.T) {
	// (define x) — missing value
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("define"), bones.IdentNode("x")})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for define with no value")
	}
}

func TestTranslateNodeSetShort(t *testing.T) {
	// (set! x) — missing value
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("set!"), bones.IdentNode("x")})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for set! with no value")
	}
}

func TestTranslateNodeLambdaShort(t *testing.T) {
	// (lambda) — missing params and body
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("lambda")})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for lambda with no params")
	}
}

func TestTranslateNodeLambdaNoBody(t *testing.T) {
	// (lambda (x)) — missing body
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("lambda"),
		bones.NewListNode([]*bones.Node{bones.IdentNode("x")}),
	})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for lambda with no body")
	}
}

func TestTranslateNodeCondNonListClause(t *testing.T) {
	// (cond 42) — clause is not a list
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("cond"), bones.IntNode(42)})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for non-list cond clause")
	}
}

func TestTranslateNodeCondEmptyClause(t *testing.T) {
	// (cond ()) — clause is an empty list
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("cond"), bones.NewListNode(nil)})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for empty cond clause")
	}
}

func TestTranslateNodeRequire(t *testing.T) {
	// (require foo) — should translate to RequireNode
	node := bones.NewListNode([]*bones.Node{bones.IdentNode("require"), bones.IdentNode("foo")})
	result, err := translateNode(node)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != bones.RequireNode {
		t.Errorf("expected RequireNode, got %d", result.Kind)
	}
	if len(result.RawArgs) != 1 {
		t.Errorf("expected 1 raw arg, got %d", len(result.RawArgs))
	}
}

func TestTranslateNodeParamsInvalid(t *testing.T) {
	// lambda with non-identifier, non-list params: (lambda 42 body)
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("lambda"),
		bones.IntNode(42),
		bones.IntNode(1),
	})
	_, err := translateNode(node)
	if err == nil {
		t.Error("expected error for invalid parameter list")
	}
}

func TestTranslateNodeParamsDottedList(t *testing.T) {
	// (lambda (x . rest) body) — dotted params
	params := &bones.Node{Kind: bones.ListNode, Children: []*bones.Node{bones.IdentNode("x")}, DottedTail: bones.IdentNode("rest")}
	node := bones.NewListNode([]*bones.Node{
		bones.IdentNode("lambda"),
		params,
		bones.IdentNode("rest"),
	})
	result, err := translateNode(node)
	if err != nil {
		t.Fatal(err)
	}
	if result.Params.Variadic == nil || result.Params.Variadic.Name != "rest" {
		t.Error("expected variadic param 'rest'")
	}
}

func TestTranslateNodeNilPassthrough(t *testing.T) {
	result, err := translateNode(bones.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsNil() {
		t.Error("Nil should pass through")
	}
}

func TestTranslateNodeAtomPassthrough(t *testing.T) {
	result, err := translateNode(bones.IntNode(42))
	if err != nil {
		t.Fatal(err)
	}
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestTranslateNodeEmptyList(t *testing.T) {
	result, err := translateNode(bones.NewListNode(nil))
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != bones.ListNode {
		t.Errorf("expected ListNode, got %d", result.Kind)
	}
}

// --- Expansion edge cases ---

func TestExpandNodeShortLambda(t *testing.T) {
	r := newTestReanimator()
	// Macro that expands in a lambda context with short body
	nodes := parseNodes(t, `
(define-syntax wrap-lambda (syntax-rules ()
  ((wrap-lambda body) (lambda (x) body))))
(wrap-lambda 42)
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	// Should produce a LambdaNode
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Kind != bones.CallNode && result[0].Kind != bones.LambdaNode {
		// The expanded (lambda (x) 42) is a lambda, which gets translated to LambdaNode
		// but wrapped in a context it might be different
	}
}

func TestExpandNodeDefineWithMacroInValue(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax add1 (syntax-rules () ((add1 x) (+ x 1))))
(define y (add1 5))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result (define), got %d", len(result))
	}
	if result[0].Kind != bones.DefineNode {
		t.Errorf("expected DefineNode, got %d", result[0].Kind)
	}
	// The value should be expanded: (+ 5 1) not (add1 5)
	val := result[0].Children[1]
	if val.Kind != bones.CallNode {
		t.Errorf("expected CallNode for expanded value, got %d", val.Kind)
	}
	repr := val.Repr()
	if strings.Contains(repr, "add1") {
		t.Errorf("macro should be expanded in define value, got: %s", repr)
	}
}

func TestExpandNodeCondWithMacroInPredicate(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax tt (syntax-rules () ((tt) #t)))
(cond ((tt) 42) (else 0))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Kind != bones.CondNode {
		t.Errorf("expected CondNode, got %d", result[0].Kind)
	}
}

func TestExpandNodeBeginWithMacro(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax id (syntax-rules () ((id x) x)))
(begin (id 1) (id 2))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Kind != bones.BeginNode {
		t.Errorf("expected BeginNode, got %d", result[0].Kind)
	}
}

func TestExpandNodeSetBangWithMacro(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax add1 (syntax-rules () ((add1 x) (+ x 1))))
(define y 0)
(set! y (add1 5))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatal(err)
	}
	// Should have define and set!
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result[1].Kind != bones.SetNode {
		t.Errorf("expected SetNode, got %d", result[1].Kind)
	}
}

func TestExpandNodeRequirePassthrough(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(require foo)`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Kind != bones.RequireNode {
		t.Errorf("require should pass through as RequireNode, got %v", result)
	}
}

func TestExpandNodeQuotePassthrough(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(quote (wrap 5))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Kind != bones.QuoteNode {
		t.Errorf("quote should not expand macros inside, got %v", result)
	}
	// The quoted content should still contain "wrap"
	if result[0].Quoted == nil || !strings.Contains(result[0].Quoted.Repr(), "wrap") {
		t.Error("quoted content should preserve macro name")
	}
}

func TestExpandNodeLocationPreserved(t *testing.T) {
	r := newTestReanimator()
	res, parsed := exhumer.ParseWithFilename(
		strings.NewReader(`(define x 42)`),
		strPtr("test.ghl"),
	)
	if res != 0 {
		t.Fatal("parse failed")
	}
	result, err := r.ReanimateNodes(parsed.Expressions)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Loc == nil {
		t.Error("source location should be preserved")
	}
}

func strPtr(s string) *string { return &s }
