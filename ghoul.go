package ghoul

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"

	ev "github.com/archevel/ghoul/consume"
	"github.com/archevel/ghoul/reanimator"
	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
	"github.com/archevel/ghoul/exhumer"
)

//go:embed prelude/prelude.ghl
var preludeSource string

type Ghoul interface {
	Process(exprReader io.Reader) (*e.Node, error)
	ProcessFile(filename string) (*e.Node, error)
	ProcessWithContext(ctx context.Context, exprReader io.Reader, filename *string) (*e.Node, error)
}

// New creates a Ghoul instance with the standard prelude loaded.
func New() Ghoul {
	return NewLoggingGhoul(engraving.StandardLogger)
}

// NewBare creates a Ghoul instance without the prelude.
func NewBare() Ghoul {
	return newGhoul(engraving.StandardLogger, false)
}

func NewLoggingGhoul(logger engraving.Logger) Ghoul {
	return newGhoul(logger, true)
}

func newGhoul(logger engraving.Logger, loadPrelude bool) Ghoul {
	var markCounter uint64
	exp := reanimator.New(logger, &markCounter)
	// The evaluator shares the reanimator's environment so that bindings
	// from require (loaded during expansion) are visible at runtime.
	evaluator := ev.NewWithMarkCounter(logger, exp.EvalEnv(), &markCounter)
	g := ghoul{reanimator: exp, evaluator: evaluator}
	if loadPrelude {
		g.Process(strings.NewReader(preludeSource))
	}
	return g
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

	if filename != nil {
		g.reanimator.SetModuleState(ev.NewModuleState(*filename))
		g.reanimator.SetModuleLoader(makeModuleLoader(g.reanimator))
	}

	boneNodes, err := g.reanimator.ReanimateNodes(parsed.Expressions)
	if err != nil {
		return nil, fmt.Errorf("failed to expand macros: %w", err)
	}

	result, err := g.evaluator.ConsumeNodesWithContext(ctx, boneNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to process Lisp code: %w", err)
	}
	return result, nil
}

// makeModuleLoader creates a loader that processes Ghoul module files
// through the full pipeline: parse → reanimate → evaluate → extract exports.
func makeModuleLoader(r *reanimator.Reanimator) reanimator.ModuleLoader {
	return func(filePath string, parentReanimator *reanimator.Reanimator) (*ev.ModuleExports, error) {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", filePath, err)
		}
		defer f.Close()

		parseRes, parsed := exhumer.ParseWithFilename(f, &filePath)
		if parseRes != 0 {
			return nil, fmt.Errorf("failed to parse %s", filePath)
		}

		// Push a fresh macro scope for module isolation, then pop after
		savedScopes := parentReanimator.PushModuleScope()
		boneNodes, err := parentReanimator.ReanimateNodes(parsed.Expressions)
		macros := parentReanimator.ExportMacros()
		parentReanimator.PopModuleScope(savedScopes)
		if err != nil {
			return nil, fmt.Errorf("failed to expand macros in %s: %w", filePath, err)
		}

		// Evaluate in a module environment to get runtime exports
		moduleEnv := ev.NewModuleEnvironment(parentReanimator.EvalEnv())
		moduleEval := ev.NewWithMarkCounter(
			parentReanimator.Evaluator().Log(),
			moduleEnv,
			parentReanimator.Evaluator().MarkCounter(),
		)

		_, err = moduleEval.ConsumeNodes(boneNodes)
		if err != nil {
			return nil, err
		}

		exports := ev.ExtractExports(moduleEnv)
		exports.Macros = macros
		return exports, nil
	}
}

