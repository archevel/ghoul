package consume

import (
	"testing"

	"github.com/archevel/ghoul/bones"
)

func TestOpcodeConstants(t *testing.T) {
	// Verify opcodes are distinct
	opcodes := []byte{
		OP_CONST, OP_NIL, OP_TRUE, OP_FALSE, OP_POP,
		OP_LOAD_VAR, OP_DEFINE, OP_SET,
		OP_CALL, OP_TAIL_CALL, OP_RETURN,
		OP_JUMP, OP_JUMP_IF_FALSE,
		OP_MAKE_CLOSURE,
	}
	seen := map[byte]bool{}
	for _, op := range opcodes {
		if seen[op] {
			t.Errorf("duplicate opcode: %d", op)
		}
		seen[op] = true
	}
}

func TestWriteReadUint16(t *testing.T) {
	buf := make([]byte, 2)
	writeUint16(buf, 0, 0)
	if readUint16(buf, 0) != 0 {
		t.Errorf("expected 0, got %d", readUint16(buf, 0))
	}

	writeUint16(buf, 0, 256)
	if readUint16(buf, 0) != 256 {
		t.Errorf("expected 256, got %d", readUint16(buf, 0))
	}

	writeUint16(buf, 0, 65535)
	if readUint16(buf, 0) != 65535 {
		t.Errorf("expected 65535, got %d", readUint16(buf, 0))
	}
}

func TestCodeObjectAddConstant(t *testing.T) {
	co := &CodeObject{Name: "test"}
	idx1 := co.addConstant(bones.IntNode(42))
	idx2 := co.addConstant(bones.StrNode("hello"))
	idx3 := co.addConstant(bones.IntNode(42)) // duplicate

	if idx1 != 0 {
		t.Errorf("first constant should be index 0, got %d", idx1)
	}
	if idx2 != 1 {
		t.Errorf("second constant should be index 1, got %d", idx2)
	}
	// Duplicates get separate indices (no dedup for now)
	if idx3 != 2 {
		t.Errorf("third constant should be index 2, got %d", idx3)
	}
}

func TestCodeObjectEmit(t *testing.T) {
	co := &CodeObject{Name: "test"}
	co.emit(OP_NIL)
	co.emit(OP_TRUE)
	co.emit(OP_RETURN)

	if len(co.Code) != 3 {
		t.Fatalf("expected 3 bytes, got %d", len(co.Code))
	}
	if co.Code[0] != OP_NIL || co.Code[1] != OP_TRUE || co.Code[2] != OP_RETURN {
		t.Errorf("unexpected bytecode: %v", co.Code)
	}
}

func TestCodeObjectEmitWithOperand(t *testing.T) {
	co := &CodeObject{Name: "test"}
	co.emitWithOperand(OP_CONST, 42)

	if len(co.Code) != 3 {
		t.Fatalf("expected 3 bytes, got %d", len(co.Code))
	}
	if co.Code[0] != OP_CONST {
		t.Error("expected OP_CONST")
	}
	if readUint16(co.Code, 1) != 42 {
		t.Errorf("expected operand 42, got %d", readUint16(co.Code, 1))
	}
}

func TestCodeObjectEmitLoc(t *testing.T) {
	co := &CodeObject{Name: "test"}
	loc := &bones.SourcePosition{Ln: 5, Col: 3}

	co.emitWithLoc(OP_LOAD_VAR, loc)
	co.Code = append(co.Code, 0, 42) // operand

	if len(co.Locs) != 1 {
		t.Fatalf("expected 1 loc entry, got %d", len(co.Locs))
	}
	if co.Locs[0].StartPC != 0 || co.Locs[0].Loc != loc {
		t.Error("loc entry mismatch")
	}
}

func TestClosureData(t *testing.T) {
	code := &CodeObject{Name: "test-closure"}
	env := NewEnvironment()
	cd := &closureData{code: code, env: env}

	if cd.code.Name != "test-closure" {
		t.Error("expected test-closure")
	}
	if cd.env == nil {
		t.Error("expected non-nil env")
	}
}

func TestOpcodeName(t *testing.T) {
	if opcodeName(OP_CONST) != "OP_CONST" {
		t.Errorf("got %s", opcodeName(OP_CONST))
	}
	if opcodeName(OP_CALL) != "OP_CALL" {
		t.Errorf("got %s", opcodeName(OP_CALL))
	}
	if opcodeName(255) != "OP_UNKNOWN(255)" {
		t.Errorf("got %s", opcodeName(255))
	}
}
