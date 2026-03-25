package stdlib

import (
	"fmt"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
)

func registerTypes(env *ev.Environment) {
	env.Register("number?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		switch args.First().(type) {
		case e.Integer, e.Float:
			return e.Boolean(true), nil
		default:
			return e.Boolean(false), nil
		}
	})

	env.Register("integer?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		_, ok := args.First().(e.Integer)
		return e.Boolean(ok), nil
	})

	env.Register("float?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		_, ok := args.First().(e.Float)
		return e.Boolean(ok), nil
	})

	env.Register("string?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		_, ok := args.First().(e.String)
		return e.Boolean(ok), nil
	})

	env.Register("boolean?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		_, ok := args.First().(e.Boolean)
		return e.Boolean(ok), nil
	})

	env.Register("list?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		lst, ok := args.First().(e.List)
		if !ok {
			return e.Boolean(false), nil
		}
		// Check it's a proper list (ends in NIL)
		for lst != e.NIL {
			_, ok := lst.Tail()
			if !ok {
				return e.Boolean(false), nil
			}
			lst, _ = lst.Tail()
		}
		return e.Boolean(true), nil
	})
}

func registerConversions(env *ev.Environment) {
	env.Register("integer->float", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		val, ok := args.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("integer->float: expected integer, got %s", e.TypeName(args.First()))
		}
		return e.Float(val), nil
	})

	env.Register("float->integer", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		val, ok := args.First().(e.Float)
		if !ok {
			return nil, fmt.Errorf("float->integer: expected float, got %s", e.TypeName(args.First()))
		}
		return e.Integer(int64(val)), nil
	})
}
