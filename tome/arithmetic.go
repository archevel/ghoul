package tome

import (
	"fmt"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

// asFloat coerces an integer or float node to float64.
func asFloat(node *e.Node) (float64, bool) {
	switch node.Kind {
	case e.IntegerNode:
		return float64(node.IntVal), true
	case e.FloatNodeKind:
		return node.FloatVal, true
	}
	return 0, false
}

// numResult returns an IntNode if val is a whole number and allInt is true,
// otherwise a FloatNode.
func numResult(val float64, allInt bool) *e.Node {
	if allInt {
		return e.IntNode(int64(val))
	}
	return e.FloatNode(val)
}

func registerArithmetic(env *ev.Environment) {
	// (+) → 0, (+ a) → a, (+ a b ...) → sum
	env.Register("+", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		acc := 0.0
		allInt := true
		for _, arg := range args {
			v, ok := asFloat(arg)
			if !ok {
				return nil, fmt.Errorf("+: expected number, got %s", e.NodeTypeName(arg))
			}
			if arg.Kind == e.FloatNodeKind {
				allInt = false
			}
			acc += v
		}
		return numResult(acc, allInt), nil
	})

	// (*) → 1, (* a) → a, (* a b ...) → product
	env.Register("*", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		acc := 1.0
		allInt := true
		for _, arg := range args {
			v, ok := asFloat(arg)
			if !ok {
				return nil, fmt.Errorf("*: expected number, got %s", e.NodeTypeName(arg))
			}
			if arg.Kind == e.FloatNodeKind {
				allInt = false
			}
			acc *= v
		}
		return numResult(acc, allInt), nil
	})

	// (- a) → negation, (- a b ...) → a - b - ...
	env.Register("-", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("-: expected at least one argument")
		}
		first, ok := asFloat(args[0])
		if !ok {
			return nil, fmt.Errorf("-: expected number, got %s", e.NodeTypeName(args[0]))
		}
		allInt := args[0].Kind == e.IntegerNode
		if len(args) == 1 {
			return numResult(-first, allInt), nil
		}
		acc := first
		for _, arg := range args[1:] {
			v, ok := asFloat(arg)
			if !ok {
				return nil, fmt.Errorf("-: expected number, got %s", e.NodeTypeName(arg))
			}
			if arg.Kind == e.FloatNodeKind {
				allInt = false
			}
			acc -= v
		}
		return numResult(acc, allInt), nil
	})

	// (/ a) → 1/a, (/ a b ...) → a / b / ...
	env.Register("/", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("/: expected at least one argument")
		}
		first, ok := asFloat(args[0])
		if !ok {
			return nil, fmt.Errorf("/: expected number, got %s", e.NodeTypeName(args[0]))
		}
		allInt := args[0].Kind == e.IntegerNode
		if len(args) == 1 {
			if first == 0 {
				return nil, fmt.Errorf("/: division by zero")
			}
			return numResult(1.0/first, false), nil
		}
		acc := first
		for _, arg := range args[1:] {
			v, ok := asFloat(arg)
			if !ok {
				return nil, fmt.Errorf("/: expected number, got %s", e.NodeTypeName(arg))
			}
			if v == 0 {
				return nil, fmt.Errorf("/: division by zero")
			}
			if arg.Kind == e.FloatNodeKind {
				allInt = false
			}
			acc /= v
		}
		return numResult(acc, allInt), nil
	})

	// mod stays binary
	env.Register("mod", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("mod: expected 2 arguments, got %d", len(args))
		}
		a, b := args[0], args[1]
		if a.Kind != e.IntegerNode {
			return nil, fmt.Errorf("mod: expected integer as first argument, got %s", e.NodeTypeName(a))
		}
		if b.Kind != e.IntegerNode {
			return nil, fmt.Errorf("mod: expected integer as second argument, got %s", e.NodeTypeName(b))
		}
		if b.IntVal == 0 {
			return nil, fmt.Errorf("mod: division by zero")
		}
		return e.IntNode(a.IntVal % b.IntVal), nil
	})
}
