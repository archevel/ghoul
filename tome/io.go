package tome

import (
	"fmt"

	ev "github.com/archevel/ghoul/consume"
	e "github.com/archevel/ghoul/bones"
)

func registerIO(env *ev.Environment) {
	env.Register("println", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fst, ok := args.First().(e.String)
		if ok {
			fmt.Println(fst)
		} else {
			fmt.Println(args.First().Repr())
		}
		return e.NIL, nil
	})

	env.Register("print", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fst, ok := args.First().(e.String)
		if ok {
			fmt.Print(fst)
		} else {
			fmt.Print(args.First().Repr())
		}
		return e.NIL, nil
	})
}
