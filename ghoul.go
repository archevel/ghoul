package ghoul

import (
	"context"
	"fmt"
	"io"
	"os"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
	"github.com/archevel/ghoul/parser"
	"github.com/archevel/ghoul/stdlib"
)

type Ghoul interface {
	Process(exprReader io.Reader) (e.Expr, error)
	ProcessFile(filename string) (e.Expr, error)
	ProcessWithContext(ctx context.Context, exprReader io.Reader, filename *string) (e.Expr, error)
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

	if filename != nil {
		g.evaluator.SetModuleState(ev.NewModuleState(*filename))
	}

	result, err := g.evaluator.EvaluateWithContext(ctx, parsed.Expressions)
	if err != nil {
		return nil, fmt.Errorf("failed to process Lisp code: %w", err)
	}
	return result, nil
}

func prepareEvaluator(logger logging.Logger) *ev.Evaluator {
	env := ev.NewEnvironment()
	stdlib.RegisterAll(env)
	return ev.New(logger, env)
}
