package ghoul

import (
	"context"
	"fmt"
	"io"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
	m "github.com/archevel/ghoul/macromancy"
	"github.com/archevel/ghoul/parser"
)

type Ghoul interface {
	Process(exprReader io.Reader) (e.Expr, error)
	ProcessWithContext(ctx context.Context, exprReader io.Reader) (e.Expr, error)
}

func New() Ghoul {
	return NewLoggingGhoul(logging.StandardLogger) // Less verbose by default
}

func NewLoggingGhoul(logger logging.Logger) Ghoul {
	evaluator := prepareEvaluator(logger)
	mancer := m.NewMacromancer(logger)
	return ghoul{evaluator, mancer}
}

type ghoul struct {
	evaluator   *ev.Evaluator
	macromancer *m.Macromancer
}

func (g ghoul) Process(exprReader io.Reader) (e.Expr, error) {
	return g.ProcessWithContext(context.Background(), exprReader)
}

func (g ghoul) ProcessWithContext(ctx context.Context, exprReader io.Reader) (e.Expr, error) {
	parseRes, parsed := parser.Parse(exprReader)
	if parseRes != 0 {
		return nil, fmt.Errorf("failed to parse Lisp code: parse result %d", parseRes)
	}

	manced := g.macromancer.Transform(parsed.Expressions)
	result, err := g.evaluator.EvaluateWithContext(ctx, manced)

	return result, err
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
			return nil, fmt.Errorf("and: first argument must be boolean, got %T", args.First())
		}
		t, _ := args.Tail()
		snd, ok := t.First().(e.Boolean)
		if !ok {
			return nil, fmt.Errorf("and: second argument must be boolean, got %T", t.First())
		}
		return e.Boolean(fst && snd), nil
	})

	env.Register("<", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst, ok := args.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("<: first argument must be integer, got %T", args.First())
		}
		t, _ := args.Tail()
		snd, ok := t.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("<: second argument must be integer, got %T", t.First())
		}
		return e.Boolean(fst < snd), nil
	})

	env.Register("mod", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst, ok := args.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("mod: first argument must be integer, got %T", args.First())
		}
		t, _ := args.Tail()
		snd, ok := t.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("mod: second argument must be integer, got %T", t.First())
		}
		if snd == 0 {
			return nil, fmt.Errorf("mod: division by zero")
		}
		return e.Integer(fst % snd), nil
	})

	env.Register("+", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst, ok := args.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("+: first argument must be integer, got %T", args.First())
		}
		t, _ := args.Tail()
		snd, ok := t.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("+: second argument must be integer, got %T", t.First())
		}
		return e.Integer(fst + snd), nil
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
