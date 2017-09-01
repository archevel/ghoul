package ghoul

import (
	"fmt"
	"io"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	m "github.com/archevel/ghoul/macromancy"
	parser "github.com/archevel/ghoul/parser"
)

type Ghoul interface {
	Process(exprReader io.Reader) (e.Expr, error)
}

func NewGhoul() Ghoul {
	evaluator := prepareEvaluator()
	mancer := m.NewMacromancer()
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

func prepareEvaluator() *ev.Evaluator {
	env := ev.NewEnvironment()

	env.Register("eq?", func(args e.List) (e.Expr, error) {
		fst := args.Head()
		t, _ := args.Tail().(e.List)
		snd := t.Head()
		return e.Boolean(fst.Equiv(snd)), nil
	})

	env.Register("and", func(args e.List) (e.Expr, error) {
		fst := args.Head().(e.Boolean)
		t, _ := args.Tail().(e.List)
		snd := t.Head().(e.Boolean)
		return e.Boolean(fst && snd), nil
	})

	env.Register("<", func(args e.List) (e.Expr, error) {
		fst := args.Head().(e.Integer)
		t, _ := args.Tail().(e.List)
		snd := t.Head().(e.Integer)
		return e.Boolean(fst < snd), nil
	})

	env.Register("mod", func(args e.List) (e.Expr, error) {
		fst := args.Head().(e.Integer)
		t, _ := args.Tail().(e.List)
		snd := t.Head().(e.Integer)
		return e.Integer(fst % snd), nil
	})

	env.Register("+", func(args e.List) (e.Expr, error) {
		fst := args.Head().(e.Integer)
		t, _ := args.Tail().(e.List)
		snd := t.Head().(e.Integer)
		return e.Integer(fst + snd), nil
	})

	env.Register("println", func(args e.List) (e.Expr, error) {
		fst, ok := args.Head().(e.String)
		if ok {
			fmt.Println(fst)
		} else {
			fmt.Println(args.Head().Repr())
		}
		return e.NIL, nil
	})
	return ev.NewEvaluator(env)
}
