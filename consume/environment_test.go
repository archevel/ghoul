package consume

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestNewEnvironmentHasOneScope(t *testing.T) {
	env := NewEnvironment()

	scopeCount := len(*env)

	if scopeCount != 1 {
		t.Errorf("Expected scope count to be 1 was %d", scopeCount)
	}
}

func TestBoundFunctionsCanBeFoundByTheirId(t *testing.T) {
	env := NewEnvironment()
	nilFunc := func(args []*e.Node, ev *Evaluator) (*e.Node, error) { return e.Nil, nil }
	env.Register("foo", nilFunc)

	val, err := env.LookupByName("foo")
	if err != nil {
		t.Fatal(err)
	}
	if val.Kind != e.FunctionNode {
		t.Errorf("expected FunctionNode, got %d", val.Kind)
	}
}

func TestScopedIdentifierBindAndLookup(t *testing.T) {
	env := NewEnvironment()

	si := e.ScopedIdentNode("x", map[uint64]bool{1: true})
	_, err := bindNode(si, e.IntNode(42), env)
	if err != nil {
		t.Fatalf("unexpected error binding ScopedIdentifier: %v", err)
	}

	result, err := lookupNode(si, env)
	if err != nil {
		t.Fatalf("unexpected error looking up ScopedIdentifier: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestScopedIdentifierDoesNotConflictWithPlainIdentifier(t *testing.T) {
	env := NewEnvironment()

	plain := e.IdentNode("x")
	scoped := e.ScopedIdentNode("x", map[uint64]bool{1: true})

	bindNode(plain, e.IntNode(1), env)
	bindNode(scoped, e.IntNode(2), env)

	plainResult, _ := lookupNode(plain, env)
	scopedResult, _ := lookupNode(scoped, env)

	if !plainResult.Equiv(e.IntNode(1)) {
		t.Errorf("plain x should be 1, got %s", plainResult.Repr())
	}
	if !scopedResult.Equiv(e.IntNode(2)) {
		t.Errorf("scoped x should be 2, got %s", scopedResult.Repr())
	}
}

func TestScopedIdentifierWithEmptyMarksMatchesPlainIdentifier(t *testing.T) {
	env := NewEnvironment()

	plain := e.IdentNode("x")
	bindNode(plain, e.IntNode(42), env)

	siEmpty := e.ScopedIdentNode("x", map[uint64]bool{})
	result, err := lookupNode(siEmpty, env)
	if err != nil {
		t.Fatalf("ScopedIdentifier with empty marks should find plain binding: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestScopedIdentifierAssignment(t *testing.T) {
	env := NewEnvironment()

	si := e.ScopedIdentNode("x", map[uint64]bool{1: true})
	bindNode(si, e.IntNode(10), env)

	_, err := assignByName(si, e.IntNode(20), env)
	if err != nil {
		t.Fatalf("unexpected error assigning ScopedIdentifier: %v", err)
	}

	result, _ := lookupNode(si, env)
	if !result.Equiv(e.IntNode(20)) {
		t.Errorf("expected 20 after assignment, got %s", result.Repr())
	}
}

func TestBoundFunctionsResideInBottomScope(t *testing.T) {
	env := NewEnvironment()
	// Add a scope
	env = newEnvWithEmptyScope(env)

	nilFunc := func(args []*e.Node, ev *Evaluator) (*e.Node, error) { return e.Nil, nil }
	env.Register("foo", nilFunc)

	// Bottom scope should have the binding even with extra scope on top
	val, err := env.LookupByName("foo")
	if err != nil {
		t.Fatal(err)
	}
	if val.Kind != e.FunctionNode {
		t.Errorf("expected FunctionNode, got %d", val.Kind)
	}
}
