package tome

import (
	"fmt"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

func registerTypes(env *ev.Environment) {
	env.Register("number?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		switch args[0].Kind {
		case e.IntegerNode, e.FloatNodeKind:
			return e.BoolNode(true), nil
		default:
			return e.BoolNode(false), nil
		}
	})

	env.Register("integer?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		return e.BoolNode(args[0].Kind == e.IntegerNode), nil
	})

	env.Register("float?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		return e.BoolNode(args[0].Kind == e.FloatNodeKind), nil
	})

	env.Register("string?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		return e.BoolNode(args[0].Kind == e.StringNode), nil
	})

	env.Register("boolean?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		return e.BoolNode(args[0].Kind == e.BooleanNode), nil
	})

	env.Register("list?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		n := args[0]
		if n.Kind != e.ListNode && !n.IsNil() {
			return e.BoolNode(false), nil
		}
		if n.IsNil() {
			return e.BoolNode(true), nil
		}
		// Check it's a proper list (no dotted tail)
		if n.DottedTail != nil {
			return e.BoolNode(false), nil
		}
		return e.BoolNode(true), nil
	})
}

func registerConversions(env *ev.Environment) {
	env.Register("integer->float", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.IntegerNode {
			return nil, fmt.Errorf("integer->float: expected integer, got %s", e.NodeTypeName(args[0]))
		}
		return e.FloatNode(float64(args[0].IntVal)), nil
	})

	env.Register("float->integer", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.FloatNodeKind {
			return nil, fmt.Errorf("float->integer: expected float, got %s", e.NodeTypeName(args[0]))
		}
		return e.IntNode(int64(args[0].FloatVal)), nil
	})
}
