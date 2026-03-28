package consume

import (
	"context"
	"errors"
	"fmt"

	"github.com/archevel/ghoul/bones"
)

// VM executes compiled bytecode.
type VM struct {
	stack  []*bones.Node
	sp     int
	frames []callFrame
	fp     int

	// Shared evaluator state
	ev *Evaluator
}

const (
	defaultStackSize = 256
	defaultFrameSize = 64
)

func newVM(ev *Evaluator) *VM {
	return &VM{
		stack:  make([]*bones.Node, defaultStackSize),
		frames: make([]callFrame, defaultFrameSize),
		ev:     ev,
	}
}

func (vm *VM) push(val *bones.Node) {
	if vm.sp >= len(vm.stack) {
		vm.stack = append(vm.stack, make([]*bones.Node, len(vm.stack))...)
	}
	vm.stack[vm.sp] = val
	vm.sp++
}

func (vm *VM) pop() *bones.Node {
	vm.sp--
	val := vm.stack[vm.sp]
	vm.stack[vm.sp] = nil // help GC
	return val
}

func (vm *VM) peek() *bones.Node {
	return vm.stack[vm.sp-1]
}

func (vm *VM) run(ctx context.Context, code *CodeObject) (*bones.Node, error) {
	// Check for pre-cancelled context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Set up initial frame
	vm.frames[0] = callFrame{
		code: code,
		ip:   0,
		bp:   0,
		env:  vm.ev.env,
	}
	vm.fp = 0
	vm.sp = 0

	var counter int

	for {
		frame := &vm.frames[vm.fp]

		// Context cancellation check (every 1024 iterations)
		counter++
		if counter&0x3FF == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		if frame.ip >= len(frame.code.Code) {
			// End of code — return top of stack or Nil
			if vm.sp > 0 {
				return vm.pop(), nil
			}
			return bones.Nil, nil
		}

		op := frame.code.Code[frame.ip]
		frame.ip++

		switch op {
		case OP_CONST:
			idx := readUint16(frame.code.Code, frame.ip)
			frame.ip += 2
			vm.push(frame.code.Constants[idx])

		case OP_NIL:
			vm.push(bones.Nil)

		case OP_TRUE:
			vm.push(bones.BoolNode(true))

		case OP_FALSE:
			vm.push(bones.BoolNode(false))

		case OP_POP:
			vm.pop()

		case OP_LOAD_VAR:
			idx := readUint16(frame.code.Code, frame.ip)
			frame.ip += 2
			identNode := frame.code.Constants[idx]
			val, err := lookupNode(identNode, frame.env)
			if err != nil {
				return nil, vm.wrapError(err, frame)
			}
			vm.push(val)

		case OP_DEFINE:
			idx := readUint16(frame.code.Code, frame.ip)
			frame.ip += 2
			val := vm.peek() // leave value on stack
			nameNode := frame.code.Constants[idx]
			if _, err := bindNode(nameNode, val, frame.env); err != nil {
				return nil, vm.wrapError(err, frame)
			}

		case OP_SET:
			idx := readUint16(frame.code.Code, frame.ip)
			frame.ip += 2
			val := vm.peek()
			nameNode := frame.code.Constants[idx]
			if _, err := assignByName(nameNode, val, frame.env); err != nil {
				return nil, vm.wrapError(err, frame)
			}

		case OP_CALL:
			argc := int(readUint16(frame.code.Code, frame.ip))
			frame.ip += 2
			if err := vm.doCall(argc, false, frame); err != nil {
				return nil, err
			}

		case OP_TAIL_CALL:
			argc := int(readUint16(frame.code.Code, frame.ip))
			frame.ip += 2
			if err := vm.doCall(argc, true, frame); err != nil {
				return nil, err
			}

		case OP_RETURN:
			result := vm.pop()
			if vm.fp == 0 {
				return result, nil
			}
			vm.fp--
			vm.sp = vm.frames[vm.fp+1].bp
			vm.push(result)

		case OP_JUMP:
			offset := readUint16(frame.code.Code, frame.ip)
			frame.ip = int(offset)

		case OP_JUMP_IF_FALSE:
			offset := readUint16(frame.code.Code, frame.ip)
			frame.ip += 2
			val := vm.pop()
			if !vmTruthy(val) {
				frame.ip = int(offset)
			}

		case OP_MAKE_CLOSURE:
			idx := readUint16(frame.code.Code, frame.ip)
			frame.ip += 2
			codeNode := frame.code.Constants[idx]
			codeObj, ok := codeNode.ForeignVal.(*CodeObject)
			if !ok {
				return nil, fmt.Errorf("VM: expected CodeObject in constant pool")
			}
			closure := makeClosure(codeObj, frame.env)
			vm.push(closure)

		default:
			return nil, fmt.Errorf("VM: unknown opcode %d", op)
		}
	}
}

func (vm *VM) doCall(argc int, isTail bool, frame *callFrame) error {
	// Stack: [... arg0, arg1, ..., argN, func]
	funNode := vm.pop() // pop function

	// Collect args from stack
	args := make([]*bones.Node, argc)
	for i := argc - 1; i >= 0; i-- {
		args[i] = vm.pop()
	}

	// Check if it's a compiled Ghoul closure
	if funNode.Kind == bones.FunctionNode && funNode.ForeignVal != nil {
		if cd, ok := funNode.ForeignVal.(*closureData); ok {
			return vm.callClosure(cd, args, isTail, frame)
		}
	}

	// Go native function call
	if funNode.Kind == bones.FunctionNode && funNode.FuncVal != nil {
		proc := *funNode.FuncVal
		result, err := proc(args, vm.ev)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return vm.wrapError(err, frame)
		}
		vm.push(result)
		return nil
	}

	// ForeignNode wrapping something callable (legacy bridge)
	if funNode.Kind == bones.ForeignNode {
		type reprable interface{ Repr() string }
		if r, ok := funNode.ForeignVal.(reprable); ok {
			return vm.wrapError(fmt.Errorf("not a procedure: %s", r.Repr()), frame)
		}
	}

	return vm.wrapError(fmt.Errorf("not a procedure: %s", funNode.Repr()), frame)
}

func (vm *VM) callClosure(cd *closureData, args []*bones.Node, isTail bool, frame *callFrame) error {
	// Create new environment
	newEnv := newEnvWithEmptyScope(cd.env)

	// Bind parameters
	if cd.code.Params != nil {
		argIdx := 0
		for _, param := range cd.code.Params.Fixed {
			if argIdx >= len(args) {
				return fmt.Errorf("arity mismatch: too few arguments")
			}
			bindNode(param, args[argIdx], newEnv)
			argIdx++
		}
		if cd.code.Params.Variadic != nil {
			remaining := bones.NewListNode(args[argIdx:])
			bindNode(cd.code.Params.Variadic, remaining, newEnv)
		} else if argIdx < len(args) {
			return fmt.Errorf("arity mismatch: too many arguments")
		}
	}

	if isTail {
		// Reuse current frame
		frame.code = cd.code
		frame.ip = 0
		frame.env = newEnv
		vm.sp = frame.bp // trim stack
	} else {
		// Push new frame
		vm.fp++
		if vm.fp >= len(vm.frames) {
			vm.frames = append(vm.frames, callFrame{})
		}
		vm.frames[vm.fp] = callFrame{
			code: cd.code,
			ip:   0,
			bp:   vm.sp,
			env:  newEnv,
		}
	}

	return nil
}

func (vm *VM) wrapError(err error, frame *callFrame) error {
	if _, ok := err.(EvaluationError); ok {
		return err
	}
	loc := frame.code.locForPC(frame.ip - 1)
	if loc != nil {
		return EvaluationError{msg: err.Error(), Loc: loc}
	}
	return EvaluationError{msg: err.Error()}
}

func vmTruthy(n *bones.Node) bool {
	if n.IsNil() {
		return false
	}
	if n.Kind == bones.BooleanNode {
		return n.BoolVal
	}
	return true
}

// makeClosure creates a FuncNode that wraps a compiled closure.
// It stores the closureData in ForeignVal for the VM fast path,
// and provides a FuncVal wrapper for Go stdlib callback compatibility.
func makeClosure(code *CodeObject, env *environment) *bones.Node {
	cd := &closureData{code: code, env: env}

	// Build the node first so the wrapper can reference it
	node := &bones.Node{Kind: bones.FunctionNode, ForeignVal: cd}

	// The FuncVal wrapper allows Go stdlib functions to call this closure
	// through the existing FuncVal interface (e.g., when used in map/filter).
	closureNode := node // capture for wrapper closure
	wrapper := func(args []*bones.Node, evaluator bones.Evaluator) (*bones.Node, error) {
		ev := evaluator.(*Evaluator)
		children := make([]*bones.Node, 0, len(args)+1)
		children = append(children, closureNode)
		children = append(children, args...)
		callNode := &bones.Node{Kind: bones.CallNode, Children: children}
		return ev.EvalSubExpression(callNode)
	}
	node.FuncVal = &wrapper
	return node
}
