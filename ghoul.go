package ghoul

import (
	"context"
	"fmt"
	"io"
	"os"

	ev "github.com/archevel/ghoul/consume"
	"github.com/archevel/ghoul/reanimator"
	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
	"github.com/archevel/ghoul/exhumer"
	"github.com/archevel/ghoul/tome"
)

type Ghoul interface {
	Process(exprReader io.Reader) (e.Expr, error)
	ProcessFile(filename string) (e.Expr, error)
	ProcessWithContext(ctx context.Context, exprReader io.Reader, filename *string) (e.Expr, error)
}

func New() Ghoul {
	return NewLoggingGhoul(engraving.StandardLogger)
}

func NewLoggingGhoul(logger engraving.Logger) Ghoul {
	var markCounter uint64
	exp := reanimator.New(logger, &markCounter)
	evaluator := prepareEvaluator(logger, &markCounter)
	return ghoul{reanimator: exp, evaluator: evaluator}
}

type ghoul struct {
	reanimator *reanimator.Reanimator
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
	parseRes, parsed := exhumer.ParseWithFilename(exprReader, filename)
	if parseRes != 0 {
		return nil, fmt.Errorf("failed to parse Lisp code: parse result %d", parseRes)
	}

	// Phase 1: Macro expansion
	expanded, err := g.reanimator.ExpandAll(parsed.Expressions)
	if err != nil {
		return nil, fmt.Errorf("failed to expand macros: %w", err)
	}

	// Phase 2: Evaluation
	if filename != nil {
		g.evaluator.SetModuleState(ev.NewModuleState(*filename))
	}

	result, err := g.evaluator.EvaluateWithContext(ctx, expanded)
	if err != nil {
		return nil, fmt.Errorf("failed to process Lisp code: %w", err)
	}
	return result, nil
}

func prepareEvaluator(logger engraving.Logger, markCounter *uint64) *ev.Evaluator {
	env := ev.NewEnvironment()
	tome.RegisterAll(env)
	return ev.NewWithMarkCounter(logger, env, markCounter)
}
