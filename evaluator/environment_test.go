package evaluator

import (
	"testing"

	e "github.com/archevel/ghoul/expressions"
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
	id := e.Identifier("foo")
	nilFunc := func(args e.List, ev *Evaluator) (e.Expr, error) { return e.NIL, nil }
	expectedFun := Function{&nilFunc}
	bindFuncAtBottomAs(id, expectedFun, env)

	actual := (*(*env)[0])[keyFromIdentifier(id)]
	if actual != expectedFun {
		t.Errorf("expected '%s' to be bound to function '%s' but was: %q", id.Repr(), expectedFun.Repr(), actual)
	}
}

func TestScopedIdentifierBindAndLookup(t *testing.T) {
	env := NewEnvironment()

	si := e.ScopedIdentifier{Name: e.Identifier("x"), Marks: map[uint64]bool{1: true}}
	_, err := bindIdentifier(si, e.Integer(42), env)
	if err != nil {
		t.Fatalf("unexpected error binding ScopedIdentifier: %v", err)
	}

	result, err := lookupIdentifier(si, env)
	if err != nil {
		t.Fatalf("unexpected error looking up ScopedIdentifier: %v", err)
	}
	if !result.Equiv(e.Integer(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestScopedIdentifierDoesNotConflictWithPlainIdentifier(t *testing.T) {
	env := NewEnvironment()

	plain := e.Identifier("x")
	scoped := e.ScopedIdentifier{Name: e.Identifier("x"), Marks: map[uint64]bool{1: true}}

	bindIdentifier(plain, e.Integer(1), env)
	bindIdentifier(scoped, e.Integer(2), env)

	plainResult, _ := lookupIdentifier(plain, env)
	scopedResult, _ := lookupIdentifier(scoped, env)

	if !plainResult.Equiv(e.Integer(1)) {
		t.Errorf("plain x should be 1, got %s", plainResult.Repr())
	}
	if !scopedResult.Equiv(e.Integer(2)) {
		t.Errorf("scoped x should be 2, got %s", scopedResult.Repr())
	}
}

func TestScopedIdentifierWithEmptyMarksMatchesPlainIdentifier(t *testing.T) {
	env := NewEnvironment()

	plain := e.Identifier("x")
	bindIdentifier(plain, e.Integer(42), env)

	siEmpty := e.ScopedIdentifier{Name: e.Identifier("x"), Marks: map[uint64]bool{}}
	result, err := lookupIdentifier(siEmpty, env)
	if err != nil {
		t.Fatalf("ScopedIdentifier with empty marks should find plain binding: %v", err)
	}
	if !result.Equiv(e.Integer(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestScopedIdentifierAssignment(t *testing.T) {
	env := NewEnvironment()

	si := e.ScopedIdentifier{Name: e.Identifier("x"), Marks: map[uint64]bool{1: true}}
	bindIdentifier(si, e.Integer(10), env)

	_, err := assign(si, e.Integer(20), env)
	if err != nil {
		t.Fatalf("unexpected error assigning ScopedIdentifier: %v", err)
	}

	result, _ := lookupIdentifier(si, env)
	if !result.Equiv(e.Integer(20)) {
		t.Errorf("expected 20 after assignment, got %s", result.Repr())
	}
}

func TestBoundFunctionsResideInBottomScope(t *testing.T) {
	env := NewEnvironment()
	// Add a scope
	env = newEnvWithEmptyScope(env)

	id := e.Identifier("foo")
	nilFunc := func(args e.List, ev *Evaluator) (e.Expr, error) { return e.NIL, nil }
	expectedFun := Function{&nilFunc}
	bindFuncAtBottomAs(id, expectedFun, env)

	actual := (*(*env)[0])[keyFromIdentifier(id)]
	if actual != expectedFun {
		t.Errorf("expected '%s' to be bound to function '%s' but was: %q", id.Repr(), expectedFun.Repr(), actual)
	}
}
