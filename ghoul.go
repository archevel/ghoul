package ghoul

import (
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
}

func New() Ghoul {
	return NewLoggingGhoul(logging.VerboseLogger)
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
	parseRes, parsed := parser.Parse(exprReader)
	if parseRes != 0 {
		return nil, fmt.Errorf("Failed to parse code")
	}

	manced := g.macromancer.Transform(parsed.Expressions)
	result, err := g.evaluator.Evaluate(manced)

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
		fst := args.First().(e.Boolean)
		t, _ := args.Tail()
		snd := t.First().(e.Boolean)
		return e.Boolean(fst && snd), nil
	})

	env.Register("<", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Boolean(fst < snd), nil
	})

	env.Register("mod", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
		return e.Integer(fst % snd), nil
	})

	env.Register("+", func(args e.List, ev *ev.Evaluator) (e.Expr, error) {
		fst := args.First().(e.Integer)
		t, _ := args.Tail()
		snd := t.First().(e.Integer)
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
