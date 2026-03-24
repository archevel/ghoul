package ghoul

import (
	"context"
	"fmt"
	"io"
	"os"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
	"github.com/archevel/ghoul/macromancy"
	"github.com/archevel/ghoul/parser"
)

type Ghoul interface {
	Process(exprReader io.Reader) (e.Expr, error)
	ProcessFile(filename string) (e.Expr, error)
	ProcessWithContext(ctx context.Context, exprReader io.Reader, filename *string) (e.Expr, error)
	RegisterFunction(name string, fn func(args e.List, ev *ev.Evaluator) (e.Expr, error))
}

func New() Ghoul {
	return NewLoggingGhoul(logging.StandardLogger) // Less verbose by default
}

func NewLoggingGhoul(logger logging.Logger) Ghoul {
	evaluator := prepareEvaluator(logger)
	return ghoul{evaluator}
}

type ghoul struct {
	evaluator *ev.Evaluator
}

func (g ghoul) Process(exprReader io.Reader) (e.Expr, error) {
	return g.ProcessWithContext(context.Background(), exprReader, nil)
}

func (g ghoul) ProcessFile(filename string) (e.Expr, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return g.ProcessWithContext(context.Background(), f, &filename)
}

func (g ghoul) ProcessWithContext(ctx context.Context, exprReader io.Reader, filename *string) (e.Expr, error) {
	parseRes, parsed := parser.ParseWithFilename(exprReader, filename)
	if parseRes != 0 {
		return nil, fmt.Errorf("failed to parse Lisp code: parse result %d", parseRes)
	}

	result, err := g.evaluator.EvaluateWithContext(ctx, parsed.Expressions)
	if err != nil {
		return nil, fmt.Errorf("failed to process Lisp code: %w", err)
	}
	return result, nil
}

func (g ghoul) RegisterFunction(name string, fn func(args e.List, ev *ev.Evaluator) (e.Expr, error)) {
	g.evaluator.GetEnvironment().Register(name, fn)
}

func intBinOp(name string, op func(a, b e.Integer) (e.Expr, error)) func(e.List, *ev.Evaluator) (e.Expr, error) {
	return func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fst, ok := args.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("%s: expected integer as first argument, got %s", name, e.TypeName(args.First()))
		}
		t, _ := args.Tail()
		snd, ok := t.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("%s: expected integer as second argument, got %s", name, e.TypeName(t.First()))
		}
		return op(fst, snd)
	}
}

func prepareEvaluator(logger logging.Logger) *ev.Evaluator {
	env := ev.NewEnvironment()

	env.Register("eq?", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst := args.First()
		t, _ := args.Tail()
		snd := t.First()
		return e.Boolean(fst.Equiv(snd)), nil
	})

	env.Register("and", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
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

	env.Register("<", intBinOp("<", func(a, b e.Integer) (e.Expr, error) {
		return e.Boolean(a < b), nil
	}))

	env.Register("mod", intBinOp("mod", func(a, b e.Integer) (e.Expr, error) {
		if b == 0 {
			return nil, fmt.Errorf("mod: division by zero")
		}
		return e.Integer(a % b), nil
	}))

	env.Register("+", intBinOp("+", func(a, b e.Integer) (e.Expr, error) {
		return e.Integer(a + b), nil
	}))

	env.Register("syntax->datum", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if so, ok := arg.(macromancy.SyntaxObject); ok {
			return so.Datum, nil
		}
		return arg, nil
	})

	env.Register("datum->syntax", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		ctxArg := args.First()
		t, _ := args.Tail()
		datum := t.First()

		marks := macromancy.NewMarkSet()
		if so, ok := ctxArg.(macromancy.SyntaxObject); ok {
			marks = so.Marks
		}
		return macromancy.WrapExpr(datum, marks), nil
	})

	env.Register("identifier?", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if so, ok := arg.(macromancy.SyntaxObject); ok {
			_, isId := so.Datum.(e.Identifier)
			return e.Boolean(isId), nil
		}
		_, isId := arg.(e.Identifier)
		return e.Boolean(isId), nil
	})

	env.Register("car", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if list, ok := arg.(e.List); ok && list != e.NIL {
			return list.First(), nil
		}
		return nil, fmt.Errorf("car: expected a non-empty list, got %s", e.TypeName(arg))
	})

	env.Register("cdr", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if list, ok := arg.(e.List); ok && list != e.NIL {
			return list.Second(), nil
		}
		return nil, fmt.Errorf("cdr: expected a non-empty list, got %s", e.TypeName(arg))
	})

	env.Register("cons", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst := args.First()
		t, _ := args.Tail()
		snd := t.First()
		return e.Cons(fst, snd), nil
	})

	env.Register("list", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		return args, nil
	})

	env.Register("println", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst, ok := args.First().(e.String)
		if ok {
			fmt.Println(fst)
		} else {
			fmt.Println(args.First().Repr())
		}
		return e.NIL, nil
	})
	return ev.New(logger, env)
}
