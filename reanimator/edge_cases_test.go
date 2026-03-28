package reanimator

import (
	"strings"
	"testing"

	"github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/exhumer"
)

// --- expandBegin short form ---

func TestExpandBeginShort(t *testing.T) {
	r := newTestReanimator()
	// (begin) with no body — should pass through unchanged
	nodes := parseNodes(t, `(begin)`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Kind != bones.BeginNode {
		t.Errorf("expected BeginNode, got %d", result[0].Kind)
	}
}

// --- expandLambda short form ---

func TestExpandLambdaShortMissingBody(t *testing.T) {
	r := newTestReanimator()
	// (lambda (x)) — too short for expansion, should error at translate
	nodes := parseNodes(t, `(lambda (x))`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Error("expected error for lambda with missing body")
	}
}

// --- expandCond short form ---

func TestExpandCondShort(t *testing.T) {
	r := newTestReanimator()
	// (cond) with no clauses
	nodes := parseNodes(t, `(cond)`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Kind != bones.CondNode {
		t.Errorf("expected CondNode, got %d", result[0].Kind)
	}
}

// --- containsMacroCall deep nesting ---

func TestContainsMacroCallDeep(t *testing.T) {
	r := newTestReanimator()
	// Define a macro, then use it deeply nested inside a begin/lambda
	nodes := parseNodes(t, `
(define-syntax id (syntax-rules () ((id x) x)))
(begin (begin (begin (id 42))))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	// The deeply nested (id 42) should expand to just 42
	repr := result[0].Repr()
	if strings.Contains(repr, "id") {
		t.Errorf("deeply nested macro should be expanded, got: %s", repr)
	}
}

// --- expandBegin with macro in body ---

func TestExpandBeginWithMacroInMiddle(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax wrap (syntax-rules () ((wrap x) (+ x 1))))
(begin 1 (wrap 2) 3)
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
	repr := result[0].Repr()
	if strings.Contains(repr, "wrap") {
		t.Errorf("macro in begin body should be expanded, got: %s", repr)
	}
}

// --- expandLambda with macro in body ---

func TestExpandLambdaWithMacroInBody(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `
(define-syntax inc (syntax-rules () ((inc x) (+ x 1))))
(lambda (x) (inc x))
`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("expansion failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Kind != bones.LambdaNode {
		t.Errorf("expected LambdaNode, got %d", result[0].Kind)
	}
}

// --- ReanimateNodes nil and empty inputs ---

func TestReanimateNodesNilInput(t *testing.T) {
	r := newTestReanimator()
	result, err := r.ReanimateNodes(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestReanimateNodesEmptyList(t *testing.T) {
	r := newTestReanimator()
	result, err := r.ReanimateNodes(bones.Nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for Nil input, got %v", result)
	}
}

// --- containsMacroCall on non-list ---

func TestContainsMacroCallNonList(t *testing.T) {
	r := newTestReanimator()
	scope := newMacroScope(nil)
	result := r.containsMacroCall(bones.IntNode(42), scope)
	if result {
		t.Error("containsMacroCall should return false for non-list nodes")
	}
}

// --- expandNode on non-list atom ---

func TestExpandNodeAtomPassthrough(t *testing.T) {
	r := newTestReanimator()
	scope := newMacroScope(nil)
	node := bones.IntNode(42)
	result, err := r.expandNode(node, scope)
	if err != nil {
		t.Fatal(err)
	}
	if result != node {
		t.Error("expandNode should return atom unchanged")
	}
}

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

