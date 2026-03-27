package tome

import (
	"fmt"
	"testing"

	ev "github.com/archevel/ghoul/consume"
	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
)

// These tests exercise edge cases with improper lists and malformed inputs
// that can't be constructed through normal Ghoul code.

func callStdlibDirect(name string, args []*e.Node) (*e.Node, error) {
	env := ev.NewEnvironment()
	RegisterAll(env)

	val, err := env.LookupByName(name)
	if err != nil {
		return nil, err
	}
	if val.Kind != e.FunctionNode || val.FuncVal == nil {
		return nil, fmt.Errorf("expected function for %s", name)
	}
	evaluator := ev.New(engraving.StandardLogger, env)
	return (*val.FuncVal)(args, evaluator)
}

// --- Improper lists passed to list operations ---

func TestLengthImproperList(t *testing.T) {
	// (1 2 . 3)
	improper := &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(1), e.IntNode(2)}, DottedTail: e.IntNode(3)}
	_, err := callStdlibDirect("length", []*e.Node{improper})
	if err == nil {
		t.Error("expected error for improper list")
	}
}

func TestAppendImproperFirstList(t *testing.T) {
	improper := &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(1)}, DottedTail: e.IntNode(2)}
	second := e.NewListNode([]*e.Node{e.IntNode(3)})
	result, err := callStdlibDirect("append", []*e.Node{improper, second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != e.ListNode || len(result.Children) == 0 {
		t.Errorf("expected non-empty list, got %s", result.Repr())
	}
}

func TestReverseImproperList(t *testing.T) {
	improper := &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(1), e.IntNode(2)}, DottedTail: e.IntNode(3)}
	result, err := callStdlibDirect("reverse", []*e.Node{improper})
	if err != nil {
		t.Logf("reverse on improper list gave error (acceptable): %v", err)
	} else if result.Kind != e.ListNode || len(result.Children) == 0 {
		t.Errorf("expected non-empty list, got %s", result.Repr())
	}
}

func TestMapImproperList(t *testing.T) {
	improper := &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(1)}, DottedTail: e.IntNode(2)}
	idFn := func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		return args[0], nil
	}
	fn := e.FuncNode(func(args []*e.Node, evaluator e.Evaluator) (*e.Node, error) {
		return idFn(args, evaluator.(*ev.Evaluator))
	})
	result, _ := callStdlibDirect("map", []*e.Node{fn, improper})
	_ = result
}

func TestFilterImproperList(t *testing.T) {
	improper := &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(1)}, DottedTail: e.IntNode(2)}
	fn := e.FuncNode(func(args []*e.Node, evaluator e.Evaluator) (*e.Node, error) {
		return e.BoolNode(true), nil
	})
	result, _ := callStdlibDirect("filter", []*e.Node{fn, improper})
	_ = result
}

func TestFoldlImproperList(t *testing.T) {
	improper := &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(1)}, DottedTail: e.IntNode(2)}
	fn := e.FuncNode(func(args []*e.Node, evaluator e.Evaluator) (*e.Node, error) {
		return e.IntNode(args[0].IntVal + args[1].IntVal), nil
	})
	result, _ := callStdlibDirect("foldl", []*e.Node{fn, e.IntNode(0), improper})
	_ = result
}

// --- Comparison with non-numeric types ---

func TestNumericEqualityBothNonNumeric(t *testing.T) {
	_, err := callStdlibDirect("=", []*e.Node{e.StrNode("a"), e.StrNode("b")})
	if err == nil {
		t.Error("expected error for = with strings")
	}
}

func TestLessThanBothNonNumeric(t *testing.T) {
	_, err := callStdlibDirect("<", []*e.Node{e.BoolNode(true), e.BoolNode(false)})
	if err == nil {
		t.Error("expected error for < with booleans")
	}
}

func TestGreaterThanSecondArgNonNumeric(t *testing.T) {
	_, err := callStdlibDirect(">", []*e.Node{e.IntNode(1), e.StrNode("b")})
	if err == nil {
		t.Error("expected error for > with string second arg")
	}
}
