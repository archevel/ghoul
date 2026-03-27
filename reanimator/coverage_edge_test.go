package reanimator

import (
	"strings"
	"testing"

	"github.com/archevel/ghoul/bones"
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
