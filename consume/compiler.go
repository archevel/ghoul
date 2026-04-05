package consume

import (
	"fmt"

	"github.com/archevel/ghoul/bones"
)

// lexScope tracks variable-to-slot mappings at compile time for lexical addressing.
// Each lambda body gets its own lexScope; inner lambdas link to the outer via parent.
type lexScope struct {
	names  map[string]int // variable name → slot index in this scope
	count  int            // next available slot
	parent *lexScope      // enclosing scope (nil for top-level)
}

func newLexScope(parent *lexScope) *lexScope {
	return &lexScope{names: map[string]int{}, parent: parent}
}

// define allocates a new slot for the given name and returns its index.
func (ls *lexScope) define(name string) int {
	slot := ls.count
	ls.names[name] = slot
	ls.count++
	return slot
}

// resolve looks up a variable name in the lexical scope chain.
// Returns (depth, slot, found) where depth is 0 for the current scope,
// 1 for the immediate parent, etc.
func (ls *lexScope) resolve(name string) (int, int, bool) {
	depth := 0
	for s := ls; s != nil; s = s.parent {
		if slot, ok := s.names[name]; ok {
			return depth, slot, true
		}
		depth++
	}
	return 0, 0, false
}

// encodeLexAddr packs (depth, slot) into a uint16 operand: depth<<8 | slot.
// Supports depth 0-255 and slot 0-255.
func encodeLexAddr(depth, slot int) int {
	return (depth << 8) | slot
}

// decodeLexAddr unpacks a uint16 operand into (depth, slot).
func decodeLexAddr(operand uint16) (int, int) {
	return int(operand >> 8), int(operand & 0xFF)
}

// compileTopLevel compiles a sequence of top-level AST nodes into a CodeObject.
// Top-level code has no lexical scope — all variables are resolved via OP_LOAD_VAR.
func compileTopLevel(nodes []*bones.Node) (*CodeObject, error) {
	co := &CodeObject{Name: "top-level"}

	if len(nodes) == 0 {
		co.emit(OP_NIL)
		co.emit(OP_RETURN)
		return co, nil
	}

	for i, node := range nodes {
		if err := compileExpr(co, node, i == len(nodes)-1, nil); err != nil {
			return nil, err
		}
		// Discard intermediate results (all but the last)
		if i < len(nodes)-1 {
			co.emit(OP_POP)
		}
	}

	co.emit(OP_RETURN)
	return co, nil
}

// compileExpr compiles a single AST node. tailPos indicates whether the
// result of this expression goes directly to the caller (for TCO).
// ls is the current lexical scope (nil at top-level).
func compileExpr(co *CodeObject, node *bones.Node, tailPos bool, ls *lexScope) error {
	switch node.Kind {
	case bones.NilNode:
		co.emit(OP_NIL)

	case bones.IntegerNode, bones.FloatNodeKind, bones.StringNode:
		idx := co.addConstant(node)
		co.emitWithOperand(OP_CONST, idx)

	case bones.BooleanNode:
		if node.BoolVal {
			co.emit(OP_TRUE)
		} else {
			co.emit(OP_FALSE)
		}

	case bones.IdentifierNode:
		name := node.IdentName()
		if ls != nil && name != "" {
			if depth, slot, ok := ls.resolve(name); ok {
				co.emitWithLoc(OP_LOAD_LOCAL, node.Loc)
				co.Code = co.Code[:len(co.Code)-1]
				co.emitWithOperand(OP_LOAD_LOCAL, encodeLexAddr(depth, slot))
				return nil
			}
		}
		// Fall back to map-based lookup for globals/builtins
		idx := co.addConstant(node)
		co.emitWithLoc(OP_LOAD_VAR, node.Loc)
		co.Code = co.Code[:len(co.Code)-1]
		co.emitWithOperand(OP_LOAD_VAR, idx)

	case bones.QuoteNode:
		if node.Quoted == nil || node.Quoted.IsNil() {
			co.emit(OP_NIL)
		} else {
			idx := co.addConstant(node.Quoted)
			co.emitWithOperand(OP_CONST, idx)
		}

	case bones.DefineNode:
		return compileDefine(co, node, tailPos, ls)

	case bones.SetNode:
		return compileSet(co, node, tailPos, ls)

	case bones.BeginNode:
		return compileBegin(co, node, tailPos, ls)

	case bones.LambdaNode:
		return compileLambda(co, node, ls)

	case bones.CondNode:
		return compileCond(co, node, tailPos, ls)

	case bones.CallNode:
		return compileCall(co, node, tailPos, ls)

	case bones.FunctionNode, bones.ForeignNode, bones.ListNode, bones.MummyNode:
		// Self-evaluating runtime values
		idx := co.addConstant(node)
		co.emitWithOperand(OP_CONST, idx)

	default:
		return fmt.Errorf("compile: unsupported node kind %d", node.Kind)
	}

	return nil
}

func compileDefine(co *CodeObject, node *bones.Node, tailPos bool, ls *lexScope) error {
	name := node.Children[0].IdentName()

	if ls != nil && name != "" {
		// Allocate the slot before compiling the value so that recursive
		// references (e.g., (define walk (lambda ... (walk ...)))) can
		// resolve the name during compilation of the lambda body.
		slot := ls.define(name)
		if slot+1 > co.NumLocals {
			co.NumLocals = slot + 1
		}

		// Compile value (name is already in scope)
		if err := compileExpr(co, node.Children[1], false, ls); err != nil {
			return err
		}

		co.emitWithLoc(OP_DEFINE_LOCAL, node.Loc)
		co.Code = co.Code[:len(co.Code)-1]
		co.emitWithOperand(OP_DEFINE_LOCAL, slot)
		return nil
	}

	// Fall back to map-based define for top-level
	if err := compileExpr(co, node.Children[1], false, ls); err != nil {
		return err
	}
	nameIdx := co.addConstant(node.Children[0])
	co.emitWithLoc(OP_DEFINE, node.Loc)
	co.Code = co.Code[:len(co.Code)-1]
	co.emitWithOperand(OP_DEFINE, nameIdx)
	return nil
}

func compileSet(co *CodeObject, node *bones.Node, tailPos bool, ls *lexScope) error {
	if err := compileExpr(co, node.Children[1], false, ls); err != nil {
		return err
	}

	name := node.Children[0].IdentName()
	if ls != nil && name != "" {
		if depth, slot, ok := ls.resolve(name); ok {
			co.emitWithLoc(OP_SET_LOCAL, node.Loc)
			co.Code = co.Code[:len(co.Code)-1]
			co.emitWithOperand(OP_SET_LOCAL, encodeLexAddr(depth, slot))
			return nil
		}
	}

	// Fall back to map-based set for globals
	nameIdx := co.addConstant(node.Children[0])
	co.emitWithLoc(OP_SET, node.Loc)
	co.Code = co.Code[:len(co.Code)-1]
	co.emitWithOperand(OP_SET, nameIdx)
	return nil
}

func compileBegin(co *CodeObject, node *bones.Node, tailPos bool, ls *lexScope) error {
	if len(node.Children) == 0 {
		co.emit(OP_NIL)
		return nil
	}
	for i, child := range node.Children {
		isTail := tailPos && i == len(node.Children)-1
		if err := compileExpr(co, child, isTail, ls); err != nil {
			return err
		}
		if i < len(node.Children)-1 {
			co.emit(OP_POP)
		}
	}
	return nil
}

func compileLambda(co *CodeObject, node *bones.Node, parentLs *lexScope) error {
	// Create a new lexical scope for this lambda's body
	ls := newLexScope(parentLs)

	// Pre-allocate slots for parameters
	if node.Params != nil {
		for _, param := range node.Params.Fixed {
			name := param.IdentName()
			if name != "" {
				ls.define(name)
			}
		}
		if node.Params.Variadic != nil {
			name := node.Params.Variadic.IdentName()
			if name != "" {
				ls.define(name)
			}
		}
	}

	// Compile body into a child CodeObject
	child := &CodeObject{
		Name:      "lambda",
		Params:    node.Params,
		NumLocals: ls.count, // at least params
	}

	if len(node.Children) == 0 {
		child.emit(OP_NIL)
	} else {
		for i, bodyExpr := range node.Children {
			isTail := i == len(node.Children)-1
			if err := compileExpr(child, bodyExpr, isTail, ls); err != nil {
				return err
			}
			if i < len(node.Children)-1 {
				child.emit(OP_POP)
			}
		}
	}
	// NumLocals may have grown during body compilation (from define)
	if ls.count > child.NumLocals {
		child.NumLocals = ls.count
	}
	child.emit(OP_RETURN)

	// Store child CodeObject in parent's constant pool (wrapped as ForeignNode)
	codeNode := bones.ForeignNodeVal(child)
	idx := co.addConstant(codeNode)
	co.emitWithOperand(OP_MAKE_CLOSURE, idx)
	return nil
}

func compileCond(co *CodeObject, node *bones.Node, tailPos bool, ls *lexScope) error {
	if len(node.Clauses) == 0 {
		co.emit(OP_NIL)
		return nil
	}

	var endJumps []int // offsets to patch with end position
	hasElse := false

	for _, clause := range node.Clauses {
		if clause.IsElse {
			// Compile consequent body
			if err := compileCondBody(co, clause.Consequent, tailPos, ls); err != nil {
				return err
			}
			hasElse = true
			break
		}

		// Compile test
		if err := compileExpr(co, clause.Test, false, ls); err != nil {
			return err
		}

		// Jump past consequent if false
		co.emitWithOperand(OP_JUMP_IF_FALSE, 0) // placeholder
		jumpIfFalsePC := len(co.Code) - 2

		// Compile consequent body
		if err := compileCondBody(co, clause.Consequent, tailPos, ls); err != nil {
			return err
		}

		// Jump to end after consequent
		co.emitWithOperand(OP_JUMP, 0) // placeholder
		endJumps = append(endJumps, len(co.Code)-2)

		// Patch the JUMP_IF_FALSE to here
		writeUint16(co.Code, jumpIfFalsePC, uint16(len(co.Code)))
	}

	// If no else clause, emit NIL for the fallthrough case
	if !hasElse {
		co.emit(OP_NIL)
	}

	// Patch all end jumps
	endPC := len(co.Code)
	for _, offset := range endJumps {
		writeUint16(co.Code, offset, uint16(endPC))
	}

	return nil
}

func compileCondBody(co *CodeObject, body []*bones.Node, tailPos bool, ls *lexScope) error {
	if len(body) == 0 {
		co.emit(OP_NIL)
		return nil
	}
	for i, expr := range body {
		isTail := tailPos && i == len(body)-1
		if err := compileExpr(co, expr, isTail, ls); err != nil {
			return err
		}
		if i < len(body)-1 {
			co.emit(OP_POP)
		}
	}
	return nil
}

// intArithOp maps binary arithmetic operator names to specialized opcodes.
var intArithOp = map[string]byte{
	"+":  OP_INT_ADD,
	"-":  OP_INT_SUB,
	"*":  OP_INT_MUL,
	"<":  OP_INT_LT,
	"<=": OP_INT_LE,
	">":  OP_INT_GT,
	">=": OP_INT_GE,
}

func compileCall(co *CodeObject, node *bones.Node, tailPos bool, ls *lexScope) error {
	if len(node.Children) == 0 {
		return fmt.Errorf("compile: empty call")
	}

	argc := len(node.Children) - 1

	// Try to emit a specialized integer opcode for binary calls to known operators.
	if argc == 2 {
		callee := node.Children[0]
		if callee.Kind == bones.IdentifierNode {
			if op, ok := intArithOp[callee.IdentName()]; ok {
				// Compile the two arguments
				for _, arg := range node.Children[1:] {
					if err := compileExpr(co, arg, false, ls); err != nil {
						return err
					}
				}
				// Emit the specialized opcode with the callee name in the
				// constant pool so the VM can fall back to a normal call.
				nameIdx := co.addConstant(callee)
				co.emitWithLoc(op, node.Loc)
				co.Code = co.Code[:len(co.Code)-1]
				co.emitWithOperand(op, nameIdx)
				return nil
			}
		}
	}

	// Compile arguments left to right
	for _, arg := range node.Children[1:] {
		if err := compileExpr(co, arg, false, ls); err != nil {
			return err
		}
	}

	// Compile callee
	if err := compileExpr(co, node.Children[0], false, ls); err != nil {
		return err
	}

	// Emit call instruction
	if tailPos {
		co.emitWithLoc(OP_TAIL_CALL, node.Loc)
		co.Code = co.Code[:len(co.Code)-1]
		co.emitWithOperand(OP_TAIL_CALL, argc)
	} else {
		co.emitWithLoc(OP_CALL, node.Loc)
		co.Code = co.Code[:len(co.Code)-1]
		co.emitWithOperand(OP_CALL, argc)
	}

	return nil
}
