package consume

import (
	"context"

	"github.com/archevel/ghoul/bones"
)

// ConsumeNodes evaluates a sequence of bones.Node AST nodes.
func (ev *Evaluator) ConsumeNodes(nodes []*bones.Node) (*bones.Node, error) {
	return ev.ConsumeNodesWithContext(context.Background(), nodes)
}

func (ev *Evaluator) ConsumeNodesWithContext(ctx context.Context, nodes []*bones.Node) (*bones.Node, error) {
	code, err := compileTopLevel(nodes)
	if err != nil {
		return nil, err
	}
	vm := newVM(ev)
	return vm.run(ctx, code)
}

// EvalSubExpression evaluates a single Node expression using a fresh VM.
func (ev *Evaluator) EvalSubExpression(node *bones.Node) (*bones.Node, error) {
	subEval := &Evaluator{
		log:             ev.log,
		env:             ev.env,
		requiredModules: ev.requiredModules,
		moduleState:     ev.moduleState,
		markCounter:     ev.markCounter,
		moduleLoader:    ev.moduleLoader,
	}
	return subEval.ConsumeNodes([]*bones.Node{node})
}
