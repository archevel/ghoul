package mummy

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestBytesConvertsStringToByteSlice(t *testing.T) {
	args := []*e.Node{e.StrNode("hello")}
	result, err := BytesConvNode(args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != e.MummyNode {
		t.Fatalf("expected MummyNode, got %d", result.Kind)
	}
	bs, ok := result.ForeignVal.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", result.ForeignVal)
	}
	if string(bs) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(bs))
	}
}

func TestBytesRejectsNonString(t *testing.T) {
	args := []*e.Node{e.IntNode(42)}
	_, err := BytesConvNode(args, nil)
	if err == nil {
		t.Error("expected error for non-string argument")
	}
}

func TestBytesRejectsEmpty(t *testing.T) {
	_, err := BytesConvNode(nil, nil)
	if err == nil {
		t.Error("expected error for empty args")
	}
}

func TestStringFromBytesConvertsBack(t *testing.T) {
	args := []*e.Node{e.MummyNodeVal([]byte("world"), "[]byte")}
	result, err := StringFromBytesNode(args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != e.StringNode || result.StrVal != "world" {
		t.Errorf("expected \"world\", got %s", result.Repr())
	}
}

func TestStringFromBytesRejectsNonMummy(t *testing.T) {
	args := []*e.Node{e.StrNode("nope")}
	_, err := StringFromBytesNode(args, nil)
	if err == nil {
		t.Error("expected error for non-mummy argument")
	}
}

func TestStringFromBytesRejectsWrongMummyType(t *testing.T) {
	args := []*e.Node{e.MummyNodeVal(42, "int")}
	_, err := StringFromBytesNode(args, nil)
	if err == nil {
		t.Error("expected error for mummy not wrapping []byte")
	}
}

func TestStringFromBytesRejectsEmpty(t *testing.T) {
	_, err := StringFromBytesNode(nil, nil)
	if err == nil {
		t.Error("expected error for empty args")
	}
}

func TestIntSliceCreatesSlice(t *testing.T) {
	args := []*e.Node{e.IntNode(1), e.IntNode(2), e.IntNode(3)}
	result, err := IntSliceNode(args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != e.MummyNode {
		t.Fatalf("expected MummyNode, got %d", result.Kind)
	}
	s, ok := result.ForeignVal.([]int)
	if !ok {
		t.Fatalf("expected []int, got %T", result.ForeignVal)
	}
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", s)
	}
}

func TestIntSliceEmpty(t *testing.T) {
	result, err := IntSliceNode(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := result.ForeignVal.([]int)
	if !ok {
		t.Fatalf("expected []int, got %T", result.ForeignVal)
	}
	if len(s) != 0 {
		t.Errorf("expected empty slice, got %v", s)
	}
}

func TestIntSliceRejectsNonInteger(t *testing.T) {
	args := []*e.Node{e.IntNode(1), e.StrNode("bad")}
	_, err := IntSliceNode(args, nil)
	if err == nil {
		t.Error("expected error for non-integer element")
	}
}

func TestFloatSliceCreatesSlice(t *testing.T) {
	args := []*e.Node{e.FloatNode(1.5), e.FloatNode(2.5)}
	result, err := FloatSliceNode(args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := result.ForeignVal.([]float64)
	if !ok {
		t.Fatalf("expected []float64, got %T", result.ForeignVal)
	}
	if len(s) != 2 || s[0] != 1.5 || s[1] != 2.5 {
		t.Errorf("expected [1.5 2.5], got %v", s)
	}
}

func TestFloatSliceRejectsNonFloat(t *testing.T) {
	args := []*e.Node{e.IntNode(1)}
	_, err := FloatSliceNode(args, nil)
	if err == nil {
		t.Error("expected error for non-float element")
	}
}

func TestGoNilCreatesNilMummy(t *testing.T) {
	result, err := GoNilNode(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != e.MummyNode {
		t.Fatalf("expected MummyNode, got %d", result.Kind)
	}
	if result.ForeignVal != nil {
		t.Error("expected nil inside mummy")
	}
}

func TestGoNilRepr(t *testing.T) {
	result, _ := GoNilNode(nil, nil)
	if result.Repr() != "#<mummy:nil>" {
		t.Errorf("expected '#<mummy:nil>', got '%s'", result.Repr())
	}
}
