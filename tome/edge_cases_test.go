package tome

import (
	"testing"

	ev "github.com/archevel/ghoul/consume"
	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
)

// These tests exercise edge cases with improper lists and malformed inputs
// that can't be constructed through normal Ghoul code. They exist to
// document defensive behavior and ensure no panics on bad state.

// callStdlibDirect calls a registered stdlib function directly with
// pre-constructed arguments, bypassing the evaluator. This lets us
// pass improper lists that can't be constructed in Ghoul syntax.
func callStdlibDirect(name string, args e.List) (e.Expr, error) {
	env := ev.NewEnvironment()
	RegisterAll(env)

	val, err := env.LookupByName(name)
	if err != nil {
		return nil, err
	}
	fn := val.(ev.Function)
	evaluator := ev.New(engraving.StandardLogger, env)
	return (*fn.Fun)(args, evaluator)
}

// --- Improper lists passed to list operations ---

func TestLengthImproperList(t *testing.T) {
	// (1 2 . 3) — Tail() fails on the last pair
	improper := e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Integer(3)))
	_, err := callStdlibDirect("length", e.Cons(improper, e.NIL))
	if err == nil {
		t.Error("expected error for improper list")
	}
}

func TestAppendImproperFirstList(t *testing.T) {
	// append with improper first list — should collect elements
	// up to the improper tail without panicking
	improper := e.Cons(e.Integer(1), e.Integer(2))
	second := e.Cons(e.Integer(3), e.NIL)
	result, err := callStdlibDirect("append", e.Cons(improper, e.Cons(second, e.NIL)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := result.(e.List)
	if !ok || list == e.NIL {
		t.Errorf("expected non-empty list, got %v", result)
	}
}

func TestReverseImproperList(t *testing.T) {
	// Construct improper list directly and pass as a quoted arg
	// to avoid evaluation issues
	improper := e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Integer(3)))
	result, err := callStdlibDirect("reverse", e.Cons(improper, e.NIL))
	// Should not panic — collects elements until Tail() fails
	if err != nil {
		t.Logf("reverse on improper list gave error (acceptable): %v", err)
	} else {
		list, ok := result.(e.List)
		if !ok || list == e.NIL {
			t.Errorf("expected non-empty list, got %v", result)
		}
	}
}

func TestMapImproperList(t *testing.T) {
	// map with an improper list should process elements up to the
	// improper tail without panicking
	improper := e.Cons(e.Integer(1), e.Integer(2))
	// Use evalWithStdlib with a define + quoted improper list isn't possible,
	// so we just verify no panic via callStdlibDirect with a dummy function.
	// The function arg won't be callable this way, but the list iteration
	// should hit the break before trying to call it on the bad tail.
	idFn := func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		return args.First(), nil
	}
	fn := ev.Function{Fun: &idFn}
	result, _ := callStdlibDirect("map", e.Cons(fn, e.Cons(improper, e.NIL)))
	// map uses EvalSubExpression which won't work with raw Function values,
	// so this tests the list iteration path, not the callback
	_ = result
}

func TestFilterImproperList(t *testing.T) {
	improper := e.Cons(e.Integer(1), e.Integer(2))
	trueFn := func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		return e.Boolean(true), nil
	}
	fn := ev.Function{Fun: &trueFn}
	result, _ := callStdlibDirect("filter", e.Cons(fn, e.Cons(improper, e.NIL)))
	_ = result
}

func TestFoldlImproperList(t *testing.T) {
	improper := e.Cons(e.Integer(1), e.Integer(2))
	addFn := func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		a := args.First().(e.Integer)
		t, _ := args.Tail()
		b := t.First().(e.Integer)
		return e.Integer(a + b), nil
	}
	fn := ev.Function{Fun: &addFn}
	result, _ := callStdlibDirect("foldl", e.Cons(fn, e.Cons(e.Integer(0), e.Cons(improper, e.NIL))))
	_ = result
}

// --- Comparison with non-numeric types ---

func TestNumericEqualityBothNonNumeric(t *testing.T) {
	_, err := callStdlibDirect("=", e.Cons(e.String("a"), e.Cons(e.String("b"), e.NIL)))
	if err == nil {
		t.Error("expected error for = with strings")
	}
}

func TestLessThanBothNonNumeric(t *testing.T) {
	_, err := callStdlibDirect("<", e.Cons(e.Boolean(true), e.Cons(e.Boolean(false), e.NIL)))
	if err == nil {
		t.Error("expected error for < with booleans")
	}
}

func TestGreaterThanSecondArgNonNumeric(t *testing.T) {
	_, err := callStdlibDirect(">", e.Cons(e.Integer(1), e.Cons(e.String("b"), e.NIL)))
	if err == nil {
		t.Error("expected error for > with string second arg")
	}
}
