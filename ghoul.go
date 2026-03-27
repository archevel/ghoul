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
	Process(exprReader io.Reader) (*e.Node, error)
	ProcessFile(filename string) (*e.Node, error)
	ProcessWithContext(ctx context.Context, exprReader io.Reader, filename *string) (*e.Node, error)
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
	evaluator  *ev.Evaluator
}

func (g ghoul) Process(exprReader io.Reader) (*e.Node, error) {
	return g.ProcessWithContext(context.Background(), exprReader, nil)
}

func (g ghoul) ProcessFile(filename string) (*e.Node, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return g.ProcessWithContext(context.Background(), f, &filename)
}

func (g ghoul) ProcessWithContext(ctx context.Context, exprReader io.Reader, filename *string) (*e.Node, error) {
	parseRes, parsed := exhumer.ParseWithFilename(exprReader, filename)
	if parseRes != 0 {
		return nil, fmt.Errorf("failed to parse Lisp code: parse result %d", parseRes)
	}

	boneNodes, err := g.reanimator.ReanimateNodes(parsed.Expressions)
	if err != nil {
		return nil, fmt.Errorf("failed to expand macros: %w", err)
	}

	if filename != nil {
		g.evaluator.SetModuleState(ev.NewModuleState(*filename))
		g.evaluator.SetModuleLoader(g.makeBoneModuleLoader())
	}

	result, err := g.evaluator.ConsumeNodesWithContext(ctx, boneNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to process Lisp code: %w", err)
	}
	return result, nil
}

func (g ghoul) makeBoneModuleLoader() ev.ModuleLoader {
	return func(filePath string, moduleEnv *ev.Environment, state *ev.ModuleState) (*ev.ModuleExports, error) {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", filePath, err)
		}
		defer f.Close()

		parseRes, parsed := exhumer.ParseWithFilename(f, &filePath)
		if parseRes != 0 {
			return nil, fmt.Errorf("failed to parse %s", filePath)
		}

		boneNodes, err := g.reanimator.ReanimateNodes(parsed.Expressions)
		if err != nil {
			return nil, fmt.Errorf("failed to expand macros in %s: %w", filePath, err)
		}

		moduleEval := ev.NewWithMarkCounter(g.evaluator.Log(), moduleEnv, g.evaluator.MarkCounter())
		moduleEval.SetModuleState(state)
		moduleEval.SetModuleLoader(g.makeBoneModuleLoader())

		_, err = moduleEval.ConsumeNodes(boneNodes)
		if err != nil {
			return nil, err
		}

		return ev.ExtractExports(moduleEnv), nil
	}
}

func prepareEvaluator(logger engraving.Logger, markCounter *uint64) *ev.Evaluator {
	env := ev.NewEnvironment()
	tome.RegisterAll(env)
	return ev.NewWithMarkCounter(logger, env, markCounter)
}
