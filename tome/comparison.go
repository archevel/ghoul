package tome

import (
	"fmt"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

// extractNumsFromNodes extracts two numeric arguments, promoting to float if mixed.
func extractNumsFromNodes(name string, args []*e.Node) (int64, int64, float64, float64, bool, error) {
	fst := args[0]
	snd := args[1]

	fstIsInt := fst.Kind == e.IntegerNode
	fstIsFloat := fst.Kind == e.FloatNodeKind
	sndIsInt := snd.Kind == e.IntegerNode
	sndIsFloat := snd.Kind == e.FloatNodeKind

	if fstIsInt && sndIsInt {
		return fst.IntVal, snd.IntVal, 0, 0, true, nil
	}
	if fstIsFloat && sndIsFloat {
		return 0, 0, fst.FloatVal, snd.FloatVal, false, nil
	}
	if fstIsInt && sndIsFloat {
		return 0, 0, float64(fst.IntVal), snd.FloatVal, false, nil
	}
	if fstIsFloat && sndIsInt {
		return 0, 0, fst.FloatVal, float64(snd.IntVal), false, nil
	}

	if !fstIsInt && !fstIsFloat {
		return 0, 0, 0, 0, false, fmt.Errorf("%s: expected number as first argument, got %s", name, e.NodeTypeName(fst))
	}
	return 0, 0, 0, 0, false, fmt.Errorf("%s: expected number as second argument, got %s", name, e.NodeTypeName(snd))
}

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
