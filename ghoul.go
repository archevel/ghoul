package ghoul

import (
	"fmt"
	"io"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	parser "github.com/archevel/ghoul/parser"
)

type Ghoul interface {
	Process(exprReader io.Reader) (e.Expr, error)
}

func NewGhoul() Ghoul {
	evaluator := prepareEvaluator()
	return ghoul{evaluator, nil}
}

type ghoul struct {
	evaluator *ev.Evaluator
	parsed    *parser.ParsedExpressions
}

func (g ghoul) Process(exprReader io.Reader) (e.Expr, error) {
	parseRes, parsed := parser.Parse(exprReader)
	if parseRes != 0 {
		return nil, fmt.Errorf("Failed to parse code")
	}

	g.parsed = parsed
	result, err := g.evaluator.Evaluate(g.parsed.Expressions)

	return result, err
}

func prepareEvaluator() *ev.Evaluator {
	env := ev.NewEnvironment()

	ev.RegisterFuncAs("eq?", func(args e.List) (e.Expr, error) {
		fst := args.Head()
		t, _ := args.Tail().(e.List)
		snd := t.Head()
		return e.Boolean(fst.Equiv(snd)), nil
	}, env)

	ev.RegisterFuncAs("and", func(args e.List) (e.Expr, error) {
		fst := args.Head().(e.Boolean)
		t, _ := args.Tail().(e.List)
		snd := t.Head().(e.Boolean)
		return e.Boolean(fst && snd), nil
	}, env)

	ev.RegisterFuncAs("<", func(args e.List) (e.Expr, error) {
		fst := args.Head().(e.Integer)
		t, _ := args.Tail().(e.List)
		snd := t.Head().(e.Integer)
		return e.Boolean(fst < snd), nil
	}, env)

	ev.RegisterFuncAs("mod", func(args e.List) (e.Expr, error) {
		fst := args.Head().(e.Integer)
		t, _ := args.Tail().(e.List)
		snd := t.Head().(e.Integer)
		return e.Integer(fst % snd), nil
	}, env)

	ev.RegisterFuncAs("+", func(args e.List) (e.Expr, error) {
		fst := args.Head().(e.Integer)
		t, _ := args.Tail().(e.List)
		snd := t.Head().(e.Integer)
		return e.Integer(fst + snd), nil
	}, env)

	ev.RegisterFuncAs("println", func(args e.List) (e.Expr, error) {
		fst, ok := args.Head().(e.String)
		if ok {
			fmt.Println(fst)
		} else {
			fmt.Println(args.Head().Repr())
		}
		return e.NIL, nil
	}, env)
	return ev.NewEvaluator(env)
}
