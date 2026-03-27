package tome

import (
	"fmt"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

func registerIO(env *ev.Environment) {
	env.Register("println", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		fst := args[0]
		if fst.Kind == e.StringNode {
			fmt.Println(fst.StrVal)
		} else {
			fmt.Println(fst.Repr())
		}
		return e.Nil, nil
	})

	env.Register("print", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		fst := args[0]
		if fst.Kind == e.StringNode {
			fmt.Print(fst.StrVal)
		} else {
			fmt.Print(fst.Repr())
		}
		return e.Nil, nil
	})
}
