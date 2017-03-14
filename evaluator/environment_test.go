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
	nilFunc := func(args e.List) (e.Expr, error) { return e.NIL, nil }
	expectedFun := e.Function{&nilFunc}
	bindFuncAtBottomAs(id, expectedFun, env)

	actual := (*(*env)[0])[id]
	if actual != expectedFun {
		t.Errorf("expected '%s' to be bound to function '%s' but was: %q", id.Repr(), expectedFun.Repr(), actual)
	}
}

func TestBoundFunctionsResideInBottomScope(t *testing.T) {
	env := NewEnvironment()
	// Add a scope
	env = newEnvWithEmptyScope(env)

	id := e.Identifier("foo")
	nilFunc := func(args e.List) (e.Expr, error) { return e.NIL, nil }
	expectedFun := e.Function{&nilFunc}
	bindFuncAtBottomAs(id, expectedFun, env)

	actual := (*(*env)[0])[id]
	if actual != expectedFun {
		t.Errorf("expected '%s' to be bound to function '%s' but was: %q", id.Repr(), expectedFun.Repr(), actual)
	}
}
