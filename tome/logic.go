package tome

import (
	"fmt"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

func registerLogic(env *ev.Environment) {
	env.Register("and", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		fst := args[0]
		if fst.Kind != e.BooleanNode {
			return nil, fmt.Errorf("and: expected boolean as first argument, got %s", e.NodeTypeName(fst))
		}
		snd := args[1]
		if snd.Kind != e.BooleanNode {
			return nil, fmt.Errorf("and: expected boolean as second argument, got %s", e.NodeTypeName(snd))
		}
		return e.BoolNode(fst.BoolVal && snd.BoolVal), nil
	})

	env.Register("not", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		val := args[0]
		if val.Kind != e.BooleanNode {
			return nil, fmt.Errorf("not: expected boolean, got %s", e.NodeTypeName(val))
		}
		return e.BoolNode(!val.BoolVal), nil
	})
}
