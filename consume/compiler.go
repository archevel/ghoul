package consume

import (
	"fmt"

	"github.com/archevel/ghoul/bones"
)

// compileTopLevel compiles a sequence of top-level AST nodes into a CodeObject.
func compileTopLevel(nodes []*bones.Node) (*CodeObject, error) {
	co := &CodeObject{Name: "top-level"}

	if len(nodes) == 0 {
		co.emit(OP_NIL)
		co.emit(OP_RETURN)
		return co, nil
	}

	for i, node := range nodes {
		if err := compileExpr(co, node, i == len(nodes)-1); err != nil {
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
func compileExpr(co *CodeObject, node *bones.Node, tailPos bool) error {
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
		idx := co.addConstant(node)
		co.emitWithLoc(OP_LOAD_VAR, node.Loc)
		// Overwrite the emitted byte with operand
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
		return compileDefine(co, node, tailPos)

	case bones.SetNode:
		return compileSet(co, node, tailPos)

	case bones.BeginNode:
		return compileBegin(co, node, tailPos)

	case bones.LambdaNode:
		return compileLambda(co, node)

	case bones.CondNode:
		return compileCond(co, node, tailPos)

	case bones.CallNode:
		return compileCall(co, node, tailPos)

	case bones.FunctionNode, bones.ForeignNode, bones.ListNode, bones.MummyNode:
		// Self-evaluating runtime values
		idx := co.addConstant(node)
		co.emitWithOperand(OP_CONST, idx)

	default:
		return fmt.Errorf("compile: unsupported node kind %d", node.Kind)
	}

	return nil
}

func compileDefine(co *CodeObject, node *bones.Node, tailPos bool) error {
	// Compile value
	if err := compileExpr(co, node.Children[1], false); err != nil {
		return err
	}
	// Define the name
	nameIdx := co.addConstant(node.Children[0])
	co.emitWithLoc(OP_DEFINE, node.Loc)
	co.Code = co.Code[:len(co.Code)-1]
	co.emitWithOperand(OP_DEFINE, nameIdx)
	return nil
}

func compileSet(co *CodeObject, node *bones.Node, tailPos bool) error {
	if err := compileExpr(co, node.Children[1], false); err != nil {
		return err
	}
	nameIdx := co.addConstant(node.Children[0])
	co.emitWithLoc(OP_SET, node.Loc)
	co.Code = co.Code[:len(co.Code)-1]
	co.emitWithOperand(OP_SET, nameIdx)
	return nil
}

func compileBegin(co *CodeObject, node *bones.Node, tailPos bool) error {
	if len(node.Children) == 0 {
		co.emit(OP_NIL)
		return nil
	}
	for i, child := range node.Children {
		isTail := tailPos && i == len(node.Children)-1
		if err := compileExpr(co, child, isTail); err != nil {
			return err
		}
		if i < len(node.Children)-1 {
			co.emit(OP_POP)
		}
	}
	return nil
}

func compileLambda(co *CodeObject, node *bones.Node) error {
	// Compile body into a child CodeObject
	child := &CodeObject{
		Name:   "lambda",
		Params: node.Params,
	}

	if len(node.Children) == 0 {
		child.emit(OP_NIL)
	} else {
		for i, bodyExpr := range node.Children {
			isTail := i == len(node.Children)-1
			if err := compileExpr(child, bodyExpr, isTail); err != nil {
				return err
			}
			if i < len(node.Children)-1 {
				child.emit(OP_POP)
			}
		}
	}
	child.emit(OP_RETURN)

	// Store child CodeObject in parent's constant pool (wrapped as ForeignNode)
	codeNode := bones.ForeignNodeVal(child)
	idx := co.addConstant(codeNode)
	co.emitWithOperand(OP_MAKE_CLOSURE, idx)
	return nil
}

func compileCond(co *CodeObject, node *bones.Node, tailPos bool) error {
	if len(node.Clauses) == 0 {
		co.emit(OP_NIL)
		return nil
	}

	var endJumps []int // offsets to patch with end position
	hasElse := false

	for _, clause := range node.Clauses {
		if clause.IsElse {
			// Compile consequent body
			if err := compileCondBody(co, clause.Consequent, tailPos); err != nil {
				return err
			}
			hasElse = true
			break
		}

		// Compile test
		if err := compileExpr(co, clause.Test, false); err != nil {
			return err
		}

		// Jump past consequent if false
		co.emitWithOperand(OP_JUMP_IF_FALSE, 0) // placeholder
		jumpIfFalsePC := len(co.Code) - 2

		// Compile consequent body
		if err := compileCondBody(co, clause.Consequent, tailPos); err != nil {
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

func compileCondBody(co *CodeObject, body []*bones.Node, tailPos bool) error {
	if len(body) == 0 {
		co.emit(OP_NIL)
		return nil
	}
	for i, expr := range body {
		isTail := tailPos && i == len(body)-1
		if err := compileExpr(co, expr, isTail); err != nil {
			return err
		}
		if i < len(body)-1 {
			co.emit(OP_POP)
		}
	}
	return nil
}

func compileCall(co *CodeObject, node *bones.Node, tailPos bool) error {
	if len(node.Children) == 0 {
		return fmt.Errorf("compile: empty call")
	}

	argc := len(node.Children) - 1

	// Compile arguments left to right
	for _, arg := range node.Children[1:] {
		if err := compileExpr(co, arg, false); err != nil {
			return err
		}
	}

	// Compile callee
	if err := compileExpr(co, node.Children[0], false); err != nil {
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
