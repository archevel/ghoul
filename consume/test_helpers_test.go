package consume

import (
	"context"
	"os"

	"github.com/archevel/ghoul/engraving"
	p "github.com/archevel/ghoul/exhumer"
)

// testModuleLoader creates a simple module loader for tests that uses the
// old Pair-based Evaluate path directly, without the reanimator.
func testModuleLoader(parentEv *Evaluator) ModuleLoader {
	return func(filePath string, moduleEnv *environment, state *ModuleState) (*ModuleExports, error) {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		parseRes, parsed := p.ParseWithFilename(f, &filePath)
		if parseRes != 0 {
			return nil, NewEvaluationError("failed to parse "+filePath, nil)
		}

		moduleEval := NewWithMarkCounter(engraving.StandardLogger, moduleEnv, parentEv.markCounter)
		moduleEval.moduleState = state
		moduleEval.moduleLoader = testModuleLoader(moduleEval)

		_, err = moduleEval.EvaluateNode(context.Background(), parsed.Expressions)
		if err != nil {
			return nil, err
		}

		return ExtractExports(moduleEnv), nil
	}
}
