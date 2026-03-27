package consume

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
)

// --- BoundIdentifierNames ---

func TestBoundIdentifierNamesIncludesSpecialForms(t *testing.T) {
	env := NewEnvironment()
	names := env.BoundIdentifierNames()

	specialForms := []string{"cond", "else", "begin", "lambda", "define", "set!", "define-syntax", "syntax-rules", "quote", "require"}
	for _, sf := range specialForms {
		if !names[sf] {
			t.Errorf("BoundIdentifierNames should include special form %q", sf)
		}
	}
}

func TestBoundIdentifierNamesIncludesRegistered(t *testing.T) {
	env := NewEnvironment()
	env.Register("my-func", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.Nil, nil
	})
	names := env.BoundIdentifierNames()
	if !names["my-func"] {
		t.Error("BoundIdentifierNames should include registered function 'my-func'")
	}
}

func TestBoundIdentifierNamesExcludesMarkedBindings(t *testing.T) {
	env := NewEnvironment()
	// Bind a scoped identifier (with marks) — should NOT appear in BoundIdentifierNames
	scope := currentScope(env)
	marks := map[uint64]bool{1: true}
	(*scope)[keyFromNameAndMarks("scoped-var", marks)] = e.IntNode(42)

	names := env.BoundIdentifierNames()
	if names["scoped-var"] {
		t.Error("BoundIdentifierNames should exclude scoped (marked) bindings")
	}
}

// --- canonicalMarks ---

func TestCanonicalMarksOrder(t *testing.T) {
	marks := map[uint64]bool{3: true, 1: true, 2: true}
	result := canonicalMarks(marks)
	expected := "1,2,3"
	if result != expected {
		t.Errorf("canonicalMarks(%v) = %q, want %q", marks, result, expected)
	}
}

func TestCanonicalMarksEmpty(t *testing.T) {
	result := canonicalMarks(map[uint64]bool{})
	if result != "" {
		t.Errorf("canonicalMarks(empty) = %q, want empty string", result)
	}
}

func TestCanonicalMarksSingle(t *testing.T) {
	result := canonicalMarks(map[uint64]bool{42: true})
	if result != "42" {
		t.Errorf("canonicalMarks({42}) = %q, want \"42\"", result)
	}
}

// --- bindNode error path ---

func TestBindNodeNonIdentifierFails(t *testing.T) {
	env := NewEnvironment()
	_, err := bindNode(e.IntNode(42), e.IntNode(1), env)
	if err == nil {
		t.Error("bindNode with non-identifier should return an error")
	}
}

func TestBindNodeStringNodeFails(t *testing.T) {
	env := NewEnvironment()
	_, err := bindNode(e.StrNode("not-ident"), e.IntNode(1), env)
	if err == nil {
		t.Error("bindNode with string node should return an error")
	}
}

// --- translateParams edge cases ---

func TestTranslateParamsEmpty(t *testing.T) {
	// ((lambda () 1)) — empty param list, immediately called
	in := "((lambda () 1))"
	env := NewEnvironment()
	testInputGivesOutputWithinEnv(in, e.IntNode(1), env, t)
}

func TestTranslateParamsDottedListVariadic(t *testing.T) {
	// A lambda with variadic params via dotted list should capture remaining args
	in := "((lambda (x . rest) rest) 1 2 3)"
	env := NewEnvironment()
	expected := e.NewListNode([]*e.Node{e.IntNode(2), e.IntNode(3)})
	testInputGivesOutputWithinEnv(in, expected, env, t)
}

func TestTranslateParamsSingleVariadic(t *testing.T) {
	// (lambda args args) — single identifier as params means all-variadic
	in := "((lambda args args) 1 2 3)"
	env := NewEnvironment()
	expected := e.NewListNode([]*e.Node{e.IntNode(1), e.IntNode(2), e.IntNode(3)})
	testInputGivesOutputWithinEnv(in, expected, env, t)
}

// --- ConsumeNodes edge cases ---

func TestConsumeBoneNodesEmpty(t *testing.T) {
	env := NewEnvironment()
	ev := New(engraving.StandardLogger, env)
	result, err := ev.ConsumeNodes([]*e.Node{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsNil() {
		t.Errorf("expected Nil for empty nodes, got %s", result.Repr())
	}
}

// --- assignByName error paths ---

func TestAssignNodeByNameNonIdentifierFails(t *testing.T) {
	env := NewEnvironment()
	_, err := assignByName(e.IntNode(1), e.IntNode(42), env)
	if err == nil {
		t.Error("assignByName with non-identifier should return an error")
	}
}

func TestAssignNodeByNameUndefinedFails(t *testing.T) {
	env := NewEnvironment()
	_, err := assignByName(e.IdentNode("nonexistent"), e.IntNode(42), env)
	if err == nil {
		t.Error("assignByName for undefined identifier should return an error")
	}
}

// --- keyFromNode ---

func TestKeyFromNodeNonIdentifier(t *testing.T) {
	_, ok := keyFromNode(e.IntNode(42))
	if ok {
		t.Error("keyFromNode on IntNode should return false")
	}
}

func TestKeyFromNodeScopedIdentifier(t *testing.T) {
	marks := map[uint64]bool{5: true, 10: true}
	node := e.ScopedIdentNode("x", marks)
	key, ok := keyFromNode(node)
	if !ok {
		t.Fatal("keyFromNode on ScopedIdentNode should return true")
	}
	if key.Name != "x" {
		t.Errorf("expected name 'x', got %q", key.Name)
	}
	if key.MarksKey != "5,10" {
		t.Errorf("expected MarksKey '5,10', got %q", key.MarksKey)
	}
}
