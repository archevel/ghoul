package stdlib

import (
	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/macromancy"
	"github.com/archevel/ghoul/mummy"
)

func registerSyntax(env *ev.Environment) {
	env.Register("syntax->datum", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if so, ok := arg.(macromancy.SyntaxObject); ok {
			return so.Datum, nil
		}
		return arg, nil
	})

	env.Register("datum->syntax", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		ctxArg := args.First()
		t, _ := args.Tail()
		datum := t.First()
		marks := macromancy.NewMarkSet()
		if so, ok := ctxArg.(macromancy.SyntaxObject); ok {
			marks = so.Marks
		}
		return macromancy.WrapExpr(datum, marks), nil
	})

	env.Register("identifier?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if so, ok := arg.(macromancy.SyntaxObject); ok {
			_, isId := so.Datum.(e.Identifier)
			return e.Boolean(isId), nil
		}
		_, isId := arg.(e.Identifier)
		return e.Boolean(isId), nil
	})

	// Mummy conversion functions
	wrapConv := func(fn func(e.List, interface{}) (e.Expr, error)) func(e.List, *ev.Evaluator) (e.Expr, error) {
		return func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
			return fn(args, evaluator)
		}
	}
	env.Register("bytes", wrapConv(mummy.BytesConv))
	env.Register("string-from-bytes", wrapConv(mummy.StringFromBytes))
	env.Register("int-slice", wrapConv(mummy.IntSlice))
	env.Register("float-slice", wrapConv(mummy.FloatSlice))
	env.Register("go-nil", wrapConv(mummy.GoNil))
}
