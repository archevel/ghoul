package tome

import (
	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

func numCompare(name string, intCmp func(a, b int64) bool, floatCmp func(a, b float64) bool) func([]*e.Node, *ev.Evaluator) (*e.Node, error) {
	return func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		ia, ib, fa, fb, isInt, err := extractNumsFromNodes(name, args)
		if err != nil {
			return nil, err
		}
		if isInt {
			return e.BoolNode(intCmp(ia, ib)), nil
		}
		return e.BoolNode(floatCmp(fa, fb)), nil
	}
}

func registerComparison(env *ev.Environment) {
	env.Register("eq?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		fst := args[0]
		snd := args[1]
		return e.BoolNode(fst.Equiv(snd)), nil
	})

	env.Register("=", numCompare("=",
		func(a, b int64) bool { return a == b },
		func(a, b float64) bool { return a == b },
	))

	env.Register("<", numCompare("<",
		func(a, b int64) bool { return a < b },
		func(a, b float64) bool { return a < b },
	))

	env.Register(">", numCompare(">",
		func(a, b int64) bool { return a > b },
		func(a, b float64) bool { return a > b },
	))

	env.Register("<=", numCompare("<=",
		func(a, b int64) bool { return a <= b },
		func(a, b float64) bool { return a <= b },
	))

	env.Register(">=", numCompare(">=",
		func(a, b int64) bool { return a >= b },
		func(a, b float64) bool { return a >= b },
	))
}
