package consume

import (
	"testing"

	"github.com/archevel/ghoul/bones"
)

func TestCompileInteger(t *testing.T) {
	code, err := compileTopLevel([]*bones.Node{bones.IntNode(42)})
	if err != nil {
		t.Fatal(err)
	}
	// Should be: OP_CONST idx, OP_RETURN
	if len(code.Code) != 4 { // 1 + 2 (CONST + operand) + 1 (RETURN)
		t.Fatalf("expected 4 bytes, got %d: %v", len(code.Code), code.Code)
	}
	if code.Code[0] != OP_CONST {
		t.Errorf("expected OP_CONST, got %s", opcodeName(code.Code[0]))
	}
	if code.Code[3] != OP_RETURN {
		t.Errorf("expected OP_RETURN at end, got %s", opcodeName(code.Code[3]))
	}
	if len(code.Constants) != 1 || code.Constants[0].IntVal != 42 {
		t.Errorf("expected constant 42, got %v", code.Constants)
	}
}

func TestCompileNil(t *testing.T) {
	code, err := compileTopLevel([]*bones.Node{bones.Nil})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_NIL {
		t.Errorf("expected OP_NIL, got %s", opcodeName(code.Code[0]))
	}
}

func TestCompileBoolTrue(t *testing.T) {
	code, err := compileTopLevel([]*bones.Node{bones.BoolNode(true)})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_TRUE {
		t.Errorf("expected OP_TRUE, got %s", opcodeName(code.Code[0]))
	}
}

func TestCompileBoolFalse(t *testing.T) {
	code, err := compileTopLevel([]*bones.Node{bones.BoolNode(false)})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_FALSE {
		t.Errorf("expected OP_FALSE, got %s", opcodeName(code.Code[0]))
	}
}

func TestCompileString(t *testing.T) {
	code, err := compileTopLevel([]*bones.Node{bones.StrNode("hello")})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_CONST {
		t.Errorf("expected OP_CONST, got %s", opcodeName(code.Code[0]))
	}
	if code.Constants[0].StrVal != "hello" {
		t.Errorf("expected constant hello, got %s", code.Constants[0].Repr())
	}
}

func TestCompileFloat(t *testing.T) {
	code, err := compileTopLevel([]*bones.Node{bones.FloatNode(3.14)})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_CONST {
		t.Errorf("expected OP_CONST, got %s", opcodeName(code.Code[0]))
	}
}

func TestCompileIdentifier(t *testing.T) {
	code, err := compileTopLevel([]*bones.Node{bones.IdentNode("x")})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_LOAD_VAR {
		t.Errorf("expected OP_LOAD_VAR, got %s", opcodeName(code.Code[0]))
	}
	// The constant should be the identifier node itself
	idx := readUint16(code.Code, 1)
	if code.Constants[idx].Name != "x" {
		t.Errorf("expected identifier x, got %s", code.Constants[idx].Repr())
	}
}

func TestCompileScopedIdentifier(t *testing.T) {
	si := bones.ScopedIdentNode("x", map[uint64]bool{1: true})
	code, err := compileTopLevel([]*bones.Node{si})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_LOAD_VAR {
		t.Errorf("expected OP_LOAD_VAR, got %s", opcodeName(code.Code[0]))
	}
	idx := readUint16(code.Code, 1)
	if code.Constants[idx].Name != "x" || len(code.Constants[idx].Marks) == 0 {
		t.Errorf("expected scoped identifier x with marks, got %s", code.Constants[idx].Repr())
	}
}

func TestCompileQuote(t *testing.T) {
	q := &bones.Node{Kind: bones.QuoteNode, Quoted: bones.IntNode(42)}
	code, err := compileTopLevel([]*bones.Node{q})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_CONST {
		t.Errorf("expected OP_CONST for quote, got %s", opcodeName(code.Code[0]))
	}
	idx := readUint16(code.Code, 1)
	if code.Constants[idx].IntVal != 42 {
		t.Errorf("expected quoted 42, got %s", code.Constants[idx].Repr())
	}
}

func TestCompileQuoteNil(t *testing.T) {
	q := &bones.Node{Kind: bones.QuoteNode, Quoted: nil}
	code, err := compileTopLevel([]*bones.Node{q})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code[0] != OP_NIL {
		t.Errorf("expected OP_NIL for (quote ()), got %s", opcodeName(code.Code[0]))
	}
}

func TestCompileMultipleTopLevel(t *testing.T) {
	// Two expressions: 1 2 → should compile as: CONST 1, POP, CONST 2, RETURN
	nodes := []*bones.Node{bones.IntNode(1), bones.IntNode(2)}
	code, err := compileTopLevel(nodes)
	if err != nil {
		t.Fatal(err)
	}
	// OP_CONST 0, OP_POP, OP_CONST 1, OP_RETURN
	expected := []byte{OP_CONST, 0, 0, OP_POP, OP_CONST, 0, 1, OP_RETURN}
	if len(code.Code) != len(expected) {
		t.Fatalf("expected %d bytes, got %d: %v", len(expected), len(code.Code), code.Code)
	}
	for i, b := range expected {
		if code.Code[i] != b {
			t.Errorf("byte %d: expected %d, got %d", i, b, code.Code[i])
		}
	}
}

func TestCompileEmpty(t *testing.T) {
	code, err := compileTopLevel(nil)
	if err != nil {
		t.Fatal(err)
	}
	// Empty program: OP_NIL, OP_RETURN
	if len(code.Code) != 2 {
		t.Fatalf("expected 2 bytes, got %d", len(code.Code))
	}
	if code.Code[0] != OP_NIL || code.Code[1] != OP_RETURN {
		t.Errorf("expected OP_NIL OP_RETURN, got %v", code.Code)
	}
}
