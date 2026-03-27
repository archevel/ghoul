package tome

import (
	"fmt"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

func registerLists(env *ev.Environment) {
	env.Register("car", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		arg := args[0]
		if arg.Kind == e.ListNode && len(arg.Children) > 0 {
			return arg.First(), nil
		}
		if arg.IsNil() {
			return nil, fmt.Errorf("car: expected a non-empty list, got %s", e.NodeTypeName(arg))
		}
		return nil, fmt.Errorf("car: expected a non-empty list, got %s", e.NodeTypeName(arg))
	})

	env.Register("cdr", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		arg := args[0]
		if arg.Kind == e.ListNode && len(arg.Children) > 0 {
			return arg.Rest(), nil
		}
		if arg.IsNil() {
			return nil, fmt.Errorf("cdr: expected a non-empty list, got %s", e.NodeTypeName(arg))
		}
		return nil, fmt.Errorf("cdr: expected a non-empty list, got %s", e.NodeTypeName(arg))
	})

	env.Register("cons", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		fst := args[0]
		snd := args[1]
		// If snd is a list, prepend fst to it
		if snd.Kind == e.ListNode {
			children := make([]*e.Node, 0, len(snd.Children)+1)
			children = append(children, fst)
			children = append(children, snd.Children...)
			return &e.Node{Kind: e.ListNode, Children: children, DottedTail: snd.DottedTail}, nil
		}
		if snd.IsNil() {
			return e.NewListNode([]*e.Node{fst}), nil
		}
		// Improper list (dotted pair)
		return &e.Node{Kind: e.ListNode, Children: []*e.Node{fst}, DottedTail: snd}, nil
	})

	env.Register("list", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if len(args) == 0 {
			return e.Nil, nil
		}
		return e.NewListNode(args), nil
	})

	env.Register("length", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		lst := args[0]
		if lst.IsNil() {
			return e.IntNode(0), nil
		}
		if lst.Kind != e.ListNode {
			return nil, fmt.Errorf("length: expected list, got %s", e.NodeTypeName(lst))
		}
		if lst.DottedTail != nil {
			return nil, fmt.Errorf("length: improper list")
		}
		return e.IntNode(int64(len(lst.Children))), nil
	})

	env.Register("append", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		lst1 := args[0]
		lst2 := args[1]

		if lst1.IsNil() {
			return lst2, nil
		}
		if lst1.Kind != e.ListNode {
			return nil, fmt.Errorf("append: expected list as first argument, got %s", e.NodeTypeName(lst1))
		}

		// Build combined list
		if lst2.IsNil() || lst2.Kind == e.ListNode {
			var children []*e.Node
			children = append(children, lst1.Children...)
			if lst2.Kind == e.ListNode {
				children = append(children, lst2.Children...)
			}
			var dt *e.Node
			if lst2.Kind == e.ListNode {
				dt = lst2.DottedTail
			}
			return &e.Node{Kind: e.ListNode, Children: children, DottedTail: dt}, nil
		}
		// lst2 is not a list — becomes dotted tail
		children := make([]*e.Node, len(lst1.Children))
		copy(children, lst1.Children)
		return &e.Node{Kind: e.ListNode, Children: children, DottedTail: lst2}, nil
	})

	env.Register("reverse", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		lst := args[0]
		if lst.IsNil() {
			return e.Nil, nil
		}
		if lst.Kind != e.ListNode {
			return nil, fmt.Errorf("reverse: expected list, got %s", e.NodeTypeName(lst))
		}
		children := make([]*e.Node, len(lst.Children))
		for i, c := range lst.Children {
			children[len(lst.Children)-1-i] = c
		}
		return e.NewListNode(children), nil
	})

	env.Register("map", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		fnNode := args[0]
		lstNode := args[1]
		if !lstNode.IsNil() && lstNode.Kind != e.ListNode {
			return nil, fmt.Errorf("map: expected list as second argument, got %s", e.NodeTypeName(lstNode))
		}

		if lstNode.IsNil() {
			return e.Nil, nil
		}

		results := make([]*e.Node, 0, len(lstNode.Children))
		for _, child := range lstNode.Children {
			callNode := &e.Node{Kind: e.CallNode, Children: []*e.Node{fnNode, child}}
			result, err := evaluator.EvalSubExpression(callNode)
			if err != nil {
				return nil, fmt.Errorf("map: %w", err)
			}
			results = append(results, result)
		}

		return e.NewListNode(results), nil
	})

	env.Register("filter", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		fnNode := args[0]
		lstNode := args[1]
		if !lstNode.IsNil() && lstNode.Kind != e.ListNode {
			return nil, fmt.Errorf("filter: expected list as second argument, got %s", e.NodeTypeName(lstNode))
		}

		if lstNode.IsNil() {
			return e.Nil, nil
		}

		var results []*e.Node
		for _, child := range lstNode.Children {
			callNode := &e.Node{Kind: e.CallNode, Children: []*e.Node{fnNode, child}}
			result, err := evaluator.EvalSubExpression(callNode)
			if err != nil {
				return nil, fmt.Errorf("filter: %w", err)
			}
			if result.Kind == e.BooleanNode && result.BoolVal {
				results = append(results, child)
			}
		}

		if len(results) == 0 {
			return e.Nil, nil
		}
		return e.NewListNode(results), nil
	})

	env.Register("foldl", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		fnNode := args[0]
		acc := args[1]
		lstNode := args[2]
		if !lstNode.IsNil() && lstNode.Kind != e.ListNode {
			return nil, fmt.Errorf("foldl: expected list as third argument, got %s", e.NodeTypeName(lstNode))
		}

		if lstNode.IsNil() {
			return acc, nil
		}

		for _, child := range lstNode.Children {
			callNode := &e.Node{Kind: e.CallNode, Children: []*e.Node{fnNode, acc, child}}
			result, err := evaluator.EvalSubExpression(callNode)
			if err != nil {
				return nil, fmt.Errorf("foldl: %w", err)
			}
			acc = result
		}
		return acc, nil
	})

	env.Register("assoc", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		key := args[0]
		lstNode := args[1]
		if lstNode.IsNil() {
			return e.BoolNode(false), nil
		}
		if lstNode.Kind != e.ListNode {
			return e.BoolNode(false), nil
		}

		for _, child := range lstNode.Children {
			if child.Kind == e.ListNode && len(child.Children) > 0 && child.First().Equiv(key) {
				return child, nil
			}
		}
		return e.BoolNode(false), nil
	})

	env.Register("null?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		return e.BoolNode(args[0].IsNil()), nil
	})

	env.Register("pair?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		n := args[0]
		return e.BoolNode(n.Kind == e.ListNode && len(n.Children) > 0), nil
	})
}
