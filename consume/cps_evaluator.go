package consume

import (
	"context"
	"errors"
	"fmt"

	"github.com/archevel/ghoul/bones"
)

// ConsumeNodes evaluates a sequence of bones.Node AST nodes.
func (ev *Evaluator) ConsumeNodes(nodes []*bones.Node) (*bones.Node, error) {
	return ev.ConsumeNodesWithContext(context.Background(), nodes)
}

func (ev *Evaluator) ConsumeNodesWithContext(ctx context.Context, nodes []*bones.Node) (*bones.Node, error) {
	if len(nodes) == 0 {
		return bones.Nil, nil
	}
	ev.conts = &contStack{seqContinuation(nodes, false)}
	return ev.stepThroughContinuationsWithContext(ctx)
}

// EvalSubExpression evaluates a single Node expression using a fresh
// continuation stack.
func (ev *Evaluator) EvalSubExpression(node *bones.Node) (*bones.Node, error) {
	subEval := &Evaluator{
		log:             ev.log,
		env:             ev.env,
		requiredModules: ev.requiredModules,
		moduleState:     ev.moduleState,
		markCounter:     ev.markCounter,
	}
	return subEval.ConsumeNodes([]*bones.Node{node})
}

func seqContinuation(nodes []*bones.Node, maybeTailCall bool) continuation {
	return func(arg *bones.Node, ev *Evaluator) (*bones.Node, error) {
		if len(nodes) == 0 {
			return bones.Nil, nil
		}
		if len(nodes) > 1 {
			ev.pushContinuation(seqContinuation(nodes[1:], maybeTailCall))
		}
		isTail := maybeTailCall && len(nodes) == 1
		ev.pushContinuation(evalContinuation(nodes[0], isTail))
		return bones.Nil, nil
	}
}

func evalContinuation(node *bones.Node, maybeTailCall bool) continuation {
	return func(arg *bones.Node, ev *Evaluator) (*bones.Node, error) {
		return ev.evaluateNode(node, maybeTailCall)
	}
}

func (ev *Evaluator) evaluateNode(node *bones.Node, maybeTailCall bool) (*bones.Node, error) {
	switch node.Kind {
	case bones.NilNode:
		return bones.Nil, nil
	case bones.IntegerNode, bones.FloatNodeKind, bones.StringNode, bones.BooleanNode:
		return node, nil
	case bones.IdentifierNode:
		return ev.lookupIdent(node)
	case bones.QuoteNode:
		if node.Quoted != nil {
			return node.Quoted, nil
		}
		return bones.Nil, nil
	case bones.DefineNode:
		return ev.evaluateDefine(node, maybeTailCall)
	case bones.SetNode:
		return ev.evaluateSet(node, maybeTailCall)
	case bones.LambdaNode:
		return ev.evaluateLambda(node)
	case bones.CondNode:
		return ev.evaluateCond(node, maybeTailCall)
	case bones.BeginNode:
		ev.pushContinuation(seqContinuation(node.Children, maybeTailCall))
		return bones.Nil, nil
	case bones.CallNode:
		return ev.evaluateCall(node, maybeTailCall)
	case bones.RequireNode:
		return ev.evaluateRequire(node)
	case bones.FunctionNode:
		// Self-evaluating — FuncNode is already callable
		return node, nil
	case bones.ForeignNode:
		// Foreign nodes wrapping Functions self-evaluate to the Function
		return node, nil
	case bones.ListNode:
		// Runtime list value — self-evaluating
		return node, nil
	default:
		return node, nil
	}
}

func (ev *Evaluator) lookupIdent(node *bones.Node) (*bones.Node, error) {
	result, err := lookupNode(node, ev.env)
	if err != nil {
		return bones.Nil, nodeError(err.Error(), node)
	}
	return result, nil
}

func (ev *Evaluator) evaluateDefine(node *bones.Node, maybeTailCall bool) (*bones.Node, error) {
	nameNode := node.Children[0]
	valueNode := node.Children[1]

	ev.pushContinuation(func(value *bones.Node, ev *Evaluator) (*bones.Node, error) {
		_, err := bindNode(nameNode, value, ev.env)
		if err != nil {
			return bones.Nil, nodeError(err.Error(), node)
		}
		return value, nil
	})
	ev.pushContinuation(evalContinuation(valueNode, maybeTailCall))
	return bones.Nil, nil
}

func (ev *Evaluator) evaluateSet(node *bones.Node, maybeTailCall bool) (*bones.Node, error) {
	nameNode := node.Children[0]
	valueNode := node.Children[1]

	ev.pushContinuation(func(value *bones.Node, ev *Evaluator) (*bones.Node, error) {
		_, err := assignByName(nameNode, value, ev.env)
		if err != nil {
			return bones.Nil, nodeError(err.Error(), node)
		}
		return value, nil
	})
	ev.pushContinuation(evalContinuation(valueNode, maybeTailCall))
	return bones.Nil, nil
}

func (ev *Evaluator) evaluateLambda(node *bones.Node) (*bones.Node, error) {
	definitionEnv := ev.env
	bodyNodes := node.Children
	params := node.Params

	fun := func(args []*bones.Node, evaluator bones.Evaluator) (*bones.Node, error) {
		ev := evaluator.(*Evaluator)
		ev.pushContinuation(seqContinuation(bodyNodes, true))
		ev.pushContinuation(prepareScope(params, args, definitionEnv))
		return bones.Nil, nil
	}
	return bones.FuncNode(fun), nil
}

func prepareScope(params *bones.ParamSpec, args []*bones.Node, definitionEnv *environment) continuation {
	return func(ignore *bones.Node, ev *Evaluator) (*bones.Node, error) {
		newEnv := newEnvWithEmptyScope(definitionEnv)

		argIdx := 0
		for _, param := range params.Fixed {
			if argIdx >= len(args) {
				return bones.Nil, fmt.Errorf("arity mismatch: too few arguments")
			}
			bindNode(param, args[argIdx], newEnv)
			argIdx++
		}

		if params.Variadic != nil {
			remaining := bones.NewListNode(args[argIdx:])
			bindNode(params.Variadic, remaining, newEnv)
		} else if argIdx < len(args) {
			return bones.Nil, fmt.Errorf("arity mismatch: too many arguments")
		}

		ev.env = newEnv
		return bones.Nil, nil
	}
}


func (ev *Evaluator) evaluateCond(node *bones.Node, maybeTailCall bool) (*bones.Node, error) {
	if len(node.Clauses) == 0 {
		return bones.Nil, nil
	}
	ev.pushContinuation(condClauseContinuation(node.Clauses, 0, maybeTailCall))
	return bones.Nil, nil
}

func condClauseContinuation(clauses []*bones.CondClause, index int, maybeTailCall bool) continuation {
	return func(arg *bones.Node, ev *Evaluator) (*bones.Node, error) {
		if index >= len(clauses) {
			return bones.Nil, nil
		}
		clause := clauses[index]

		if clause.IsElse {
			ev.pushContinuation(seqContinuation(clause.Consequent, maybeTailCall))
			return bones.Nil, nil
		}

		ev.pushContinuation(func(testResult *bones.Node, ev *Evaluator) (*bones.Node, error) {
			if isTruthy(testResult) {
				ev.pushContinuation(seqContinuation(clause.Consequent, maybeTailCall))
				return bones.Nil, nil
			}
			if index+1 < len(clauses) {
				ev.pushContinuation(condClauseContinuation(clauses, index+1, maybeTailCall))
			}
			return bones.Nil, nil
		})
		ev.pushContinuation(evalContinuation(clause.Test, false))
		return bones.Nil, nil
	}
}

func isTruthy(n *bones.Node) bool {
	if n.IsNil() {
		return false
	}
	if n.Kind == bones.BooleanNode {
		return n.BoolVal
	}
	return true
}

func (ev *Evaluator) evaluateCall(node *bones.Node, maybeTailCall bool) (*bones.Node, error) {
	if len(node.Children) == 0 {
		return bones.Nil, nodeError("missing procedure expression in: ()", node)
	}

	calleeNode := node.Children[0]
	argNodes := node.Children[1:]
	callEnv := ev.env

	if !maybeTailCall {
		ev.pushContinuation(func(arg *bones.Node, ev *Evaluator) (*bones.Node, error) {
			ev.env = callEnv
			return arg, nil
		})
	}

	// Collect evaluated arguments as *bones.Node
	argSlice := make([]*bones.Node, len(argNodes))
	argIndex := len(argNodes) - 1
	collectArg := func(arg *bones.Node, ev *Evaluator) (*bones.Node, error) {
		argSlice[argIndex] = arg
		argIndex--
		return bones.Nil, nil
	}

	applyFunc := func(funVal *bones.Node, ev *Evaluator) (*bones.Node, error) {
		if funVal.Kind != bones.FunctionNode || funVal.FuncVal == nil {
			return bones.Nil, nodeError("not a procedure: "+funVal.Repr(), node)
		}
		proc := *funVal.FuncVal
		result, err := proc(argSlice, ev)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			return bones.Nil, nodeError(err.Error(), node)
		}
		return result, nil
	}

	ev.pushContinuation(applyFunc)
	ev.pushContinuation(evalContinuation(calleeNode, false))

	for i := 0; i < len(argNodes); i++ {
		ev.pushContinuation(collectArg)
		ev.pushContinuation(evalContinuation(argNodes[i], false))
	}

	return bones.Nil, nil
}

func (ev *Evaluator) evaluateRequire(node *bones.Node) (*bones.Node, error) {
	ev.pushContinuation(requireContinuation(node.RawArgs))
	return bones.Nil, nil
}

func nodeError(msg string, node *bones.Node) EvaluationError {
	if node.Loc != nil {
		return EvaluationError{msg: msg, Loc: node.Loc}
	}
	return EvaluationError{msg: msg}
}

