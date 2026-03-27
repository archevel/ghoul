package tome

import (
	"fmt"

	ev "github.com/archevel/ghoul/consume"
	e "github.com/archevel/ghoul/bones"
)

func registerLogic(env *ev.Environment) {
	env.Register("and", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fst, ok := args.First().(e.Boolean)
		if !ok {
			return nil, fmt.Errorf("and: expected boolean as first argument, got %s", e.TypeName(args.First()))
		}
		t, _ := args.Tail()
		snd, ok := t.First().(e.Boolean)
		if !ok {
			return nil, fmt.Errorf("and: expected boolean as second argument, got %s", e.TypeName(t.First()))
		}
		return e.Boolean(fst && snd), nil
	})

	env.Register("not", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		val, ok := args.First().(e.Boolean)
		if !ok {
			return nil, fmt.Errorf("not: expected boolean, got %s", e.TypeName(args.First()))
		}
		return e.Boolean(!val), nil
	})
}
