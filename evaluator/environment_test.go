package evaluator

import (
	e "github.com/archevel/ghoul/expressions"
	"testing"
)

func TestNewEnvironmentHasOneFrame(t *testing.T) {
	env := NewEnvironment()

	frameCount := len(*env)

	if frameCount != 1 {
		t.Errorf("Expected frame count to be 1 was %d", frameCount)
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

func TestBoundFunctionsResideInBottomFrame(t *testing.T) {
	env := NewEnvironment()
	// Add a frame
	env = newEnvWithEmptyFrame(env)

	id := e.Identifier("foo")
	nilFunc := func(args e.List) (e.Expr, error) { return e.NIL, nil }
	expectedFun := e.Function{&nilFunc}
	bindFuncAtBottomAs(id, expectedFun, env)

	actual := (*(*env)[0])[id]
	if actual != expectedFun {
		t.Errorf("expected '%s' to be bound to function '%s' but was: %q", id.Repr(), expectedFun.Repr(), actual)
	}
}
