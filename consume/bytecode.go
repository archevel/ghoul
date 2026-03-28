package consume

import (
	"fmt"

	"github.com/archevel/ghoul/bones"
)

// Opcodes for the Ghoul bytecode VM.
const (
	OP_CONST         byte = iota // push constants[operand]
	OP_NIL                       // push Nil
	OP_TRUE                      // push BoolNode(true)
	OP_FALSE                     // push BoolNode(false)
	OP_POP                       // discard top of stack
	OP_LOAD_VAR                  // lookup constants[operand] in env, push
	OP_DEFINE                    // pop value, bind constants[operand], push value
	OP_SET                       // pop value, assign constants[operand], push value
	OP_CALL                      // pop func + operand args, call, push result
	OP_TAIL_CALL                 // like CALL but reuse current frame
	OP_RETURN                    // pop frame, resume caller
	OP_JUMP                      // set IP to operand
	OP_JUMP_IF_FALSE             // pop, jump to operand if falsy
	OP_MAKE_CLOSURE              // create closure from CodeObject at constants[operand]
)

// CodeObject represents a compiled function or top-level script.
type CodeObject struct {
	Code      []byte          // flat bytecode stream
	Constants []*bones.Node   // constant pool
	Locs      []LocEntry      // source map: bytecode offset → source location
	Params    *bones.ParamSpec // nil for top-level scripts
	Name      string          // for debugging
}

// LocEntry maps a bytecode offset to a source location for error reporting.
type LocEntry struct {
	StartPC int
	Loc     bones.CodeLocation
}

// closureData holds a compiled function and its captured environment.
type closureData struct {
	code *CodeObject
	env  *environment
}

// callFrame tracks VM state per function call.
type callFrame struct {
	code *CodeObject
	ip   int
	bp   int          // base pointer into value stack
	env  *environment
}

// --- CodeObject helpers ---

func (co *CodeObject) addConstant(node *bones.Node) int {
	idx := len(co.Constants)
	co.Constants = append(co.Constants, node)
	return idx
}

func (co *CodeObject) emit(op byte) {
	co.Code = append(co.Code, op)
}

func (co *CodeObject) emitWithOperand(op byte, operand int) {
	co.Code = append(co.Code, op, 0, 0)
	writeUint16(co.Code, len(co.Code)-2, uint16(operand))
}

func (co *CodeObject) emitWithLoc(op byte, loc bones.CodeLocation) {
	if loc != nil {
		co.Locs = append(co.Locs, LocEntry{StartPC: len(co.Code), Loc: loc})
	}
	co.Code = append(co.Code, op)
}

// locForPC finds the source location for a given program counter.
func (co *CodeObject) locForPC(pc int) bones.CodeLocation {
	var best bones.CodeLocation
	for _, entry := range co.Locs {
		if entry.StartPC <= pc {
			best = entry.Loc
		} else {
			break
		}
	}
	return best
}

// --- Encoding helpers ---

func writeUint16(buf []byte, offset int, val uint16) {
	buf[offset] = byte(val >> 8)
	buf[offset+1] = byte(val)
}

func readUint16(buf []byte, offset int) uint16 {
	return uint16(buf[offset])<<8 | uint16(buf[offset+1])
}

// --- Debug ---

func opcodeName(op byte) string {
	switch op {
	case OP_CONST:
		return "OP_CONST"
	case OP_NIL:
		return "OP_NIL"
	case OP_TRUE:
		return "OP_TRUE"
	case OP_FALSE:
		return "OP_FALSE"
	case OP_POP:
		return "OP_POP"
	case OP_LOAD_VAR:
		return "OP_LOAD_VAR"
	case OP_DEFINE:
		return "OP_DEFINE"
	case OP_SET:
		return "OP_SET"
	case OP_CALL:
		return "OP_CALL"
	case OP_TAIL_CALL:
		return "OP_TAIL_CALL"
	case OP_RETURN:
		return "OP_RETURN"
	case OP_JUMP:
		return "OP_JUMP"
	case OP_JUMP_IF_FALSE:
		return "OP_JUMP_IF_FALSE"
	case OP_MAKE_CLOSURE:
		return "OP_MAKE_CLOSURE"
	default:
		return fmt.Sprintf("OP_UNKNOWN(%d)", op)
	}
}
