package tome

import (
	ev "github.com/archevel/ghoul/consume"
	e "github.com/archevel/ghoul/bones"
)

func numCompare(name string, intCmp func(a, b e.Integer) bool, floatCmp func(a, b e.Float) bool) func(e.List, *ev.Evaluator) (e.Expr, error) {
	return func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		ia, ib, fa, fb, isInt, err := extractNums(name, args)
		if err != nil {
			return nil, err
		}
		if isInt {
			return e.Boolean(intCmp(ia, ib)), nil
		}
		return e.Boolean(floatCmp(fa, fb)), nil
	}
}

func registerComparison(env *ev.Environment) {
	env.Register("eq?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fst := args.First()
		t, _ := args.Tail()
		snd := t.First()
		return e.Boolean(fst.Equiv(snd)), nil
	})

	env.Register("=", numCompare("=",
		func(a, b e.Integer) bool { return a == b },
		func(a, b e.Float) bool { return a == b },
	))

	env.Register("<", numCompare("<",
		func(a, b e.Integer) bool { return a < b },
		func(a, b e.Float) bool { return a < b },
	))

	env.Register(">", numCompare(">",
		func(a, b e.Integer) bool { return a > b },
		func(a, b e.Float) bool { return a > b },
	))

	env.Register("<=", numCompare("<=",
		func(a, b e.Integer) bool { return a <= b },
		func(a, b e.Float) bool { return a <= b },
	))

	env.Register(">=", numCompare(">=",
		func(a, b e.Integer) bool { return a >= b },
		func(a, b e.Float) bool { return a >= b },
	))
}
