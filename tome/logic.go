package tome

import (
	"fmt"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

func registerLogic(env *ev.Environment) {
	env.Register("not", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		val := args[0]
		if val.Kind != e.BooleanNode {
			return nil, fmt.Errorf("not: expected boolean, got %s", e.NodeTypeName(val))
		}
		return e.BoolNode(!val.BoolVal), nil
	})

}
