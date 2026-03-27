// Package consume implements a continuation-passing style (CPS) expression
// evaluator with proper tail call optimization. Evaluation proceeds by pushing
// continuations onto a stack and stepping through them in a trampoline loop.
//
// The primary entry point is ConsumeNodes which evaluates *bones.Node AST.
// The Evaluate method translates syntax Node trees
// before evaluation.
package consume

import (
	"context"
	"fmt"

	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
)

type continuation func(arg *e.Node, ev *Evaluator) (*e.Node, error)
type contStack []continuation

// Evaluate parses a *Node tree (top-level ListNode), translates to semantic
// nodes, and evaluates. This is the convenience entry point for tests.
func Evaluate(exprs *e.Node, env *environment) (*e.Node, error) {
	return EvaluateWithContext(context.Background(), exprs, env)
}

func EvaluateWithContext(ctx context.Context, exprs *e.Node, env *environment) (*e.Node, error) {
	evaluator := New(engraving.StandardLogger, env)
	return evaluator.EvaluateNode(ctx, exprs)
}

func New(logger engraving.Logger, env *environment) *Evaluator {
	var counter uint64
	return NewWithMarkCounter(logger, env, &counter)
}

// NewWithMarkCounter creates an evaluator sharing an external mark counter.
func NewWithMarkCounter(logger engraving.Logger, env *environment, markCounter *uint64) *Evaluator {
	return &Evaluator{log: logger, env: env, requiredModules: map[string]bool{}, markCounter: markCounter}
}

// ModuleLoader loads a Ghoul module file through the full pipeline
// and returns the module's exports. Injected by ghoul.go.
type ModuleLoader func(filePath string, moduleEnv *environment, state *ModuleState) (*ModuleExports, error)

type Evaluator struct {
	log             engraving.Logger
	env             *environment
	conts           *contStack
	requiredModules map[string]bool
	moduleState     *ModuleState
	markCounter     *uint64
	moduleLoader    ModuleLoader
}

// EvaluateNode translates a top-level Node tree and evaluates it.
func (ev *Evaluator) EvaluateNode(ctx context.Context, exprs *e.Node) (*e.Node, error) {
	if exprs == nil || exprs.IsNil() {
		return e.Nil, nil
	}
	var nodes []*e.Node
	if exprs.Kind == e.ListNode {
		nodes = exprs.Children
	} else {
		nodes = []*e.Node{exprs}
	}
	var translated []*e.Node
	for _, node := range nodes {
		t, err := translateForEval(node)
		if err != nil {
			return nil, err
		}
		if t.Loc == nil && node.Loc != nil {
			t.Loc = node.Loc
		}
		translated = append(translated, t)
	}
	return ev.ConsumeNodesWithContext(ctx, translated)
}

func (ev *Evaluator) SetModuleState(ms *ModuleState) {
	ev.moduleState = ms
}

func (ev *Evaluator) SetModuleLoader(loader ModuleLoader) {
	ev.moduleLoader = loader
}

func (ev *Evaluator) Log() engraving.Logger {
	return ev.log
}

func (ev *Evaluator) MarkCounter() *uint64 {
	return ev.markCounter
}

// --- Translation (ListNode → semantic nodes) ---

func translateForEval(node *e.Node) (*e.Node, error) {
	if node == nil || node.IsNil() {
		return node, nil
	}
	if node.Kind != e.ListNode || len(node.Children) == 0 {
		return node, nil
	}

	headName := node.Children[0].IdentName()
	switch headName {
	case "quote":
		if len(node.Children) < 2 {
			return nil, fmt.Errorf("bad syntax: quote requires an argument")
		}
		return &e.Node{Kind: e.QuoteNode, Quoted: node.Children[1], Loc: node.Loc}, nil
	case "define":
		if len(node.Children) < 3 {
			return nil, fmt.Errorf("bad syntax: missing value in binding")
		}
		valNode, err := translateForEval(node.Children[2])
		if err != nil {
			return nil, err
		}
		inheritLoc(node.Children[1], node)
		inheritLoc(valNode, node)
		return &e.Node{Kind: e.DefineNode, Loc: node.Loc, Children: []*e.Node{node.Children[1], valNode}}, nil
	case "set!":
		if len(node.Children) < 3 {
			return nil, fmt.Errorf("bad syntax: missing value in assignment")
		}
		valNode, err := translateForEval(node.Children[2])
		if err != nil {
			return nil, err
		}
		inheritLoc(node.Children[1], node)
		inheritLoc(valNode, node)
		return &e.Node{Kind: e.SetNode, Loc: node.Loc, Children: []*e.Node{node.Children[1], valNode}}, nil
	case "lambda":
		if len(node.Children) < 3 {
			return nil, fmt.Errorf("bad syntax: lambda requires parameters and body")
		}
		params, err := translateParams(node.Children[1])
		if err != nil {
			return nil, err
		}
		var body []*e.Node
		for _, child := range node.Children[2:] {
			t, err := translateForEval(child)
			if err != nil {
				return nil, err
			}
			inheritLoc(t, node)
			body = append(body, t)
		}
		return &e.Node{Kind: e.LambdaNode, Loc: node.Loc, Params: params, Children: body}, nil
	case "cond":
		var clauses []*e.CondClause
		for _, c := range node.Children[1:] {
			if c.Kind != e.ListNode || len(c.Children) == 0 {
				return nil, fmt.Errorf("bad syntax: cond clause must be a list")
			}
			test := c.Children[0]
			isElse := test.IdentName() == "else"
			var body []*e.Node
			for _, b := range c.Children[1:] {
				t, err := translateForEval(b)
				if err != nil {
					return nil, err
				}
				body = append(body, t)
			}
			clause := &e.CondClause{IsElse: isElse, Consequent: body}
			if !isElse {
				testNode, err := translateForEval(test)
				if err != nil {
					return nil, err
				}
				clause.Test = testNode
			}
			clauses = append(clauses, clause)
		}
		return &e.Node{Kind: e.CondNode, Loc: node.Loc, Clauses: clauses}, nil
	case "begin":
		var body []*e.Node
		for _, child := range node.Children[1:] {
			t, err := translateForEval(child)
			if err != nil {
				return nil, err
			}
			inheritLoc(t, node)
			body = append(body, t)
		}
		return &e.Node{Kind: e.BeginNode, Loc: node.Loc, Children: body}, nil
	case "require":
		return &e.Node{Kind: e.RequireNode, Loc: node.Loc, RawArgs: node.Children[1:]}, nil
	default:
		children := make([]*e.Node, len(node.Children))
		for i, child := range node.Children {
			t, err := translateForEval(child)
			if err != nil {
				return nil, err
			}
			inheritLoc(t, node)
			children[i] = t
		}
		return &e.Node{Kind: e.CallNode, Loc: node.Loc, Children: children}, nil
	}
}

func translateParams(paramNode *e.Node) (*e.ParamSpec, error) {
	if paramNode.Kind == e.IdentifierNode {
		return &e.ParamSpec{Variadic: paramNode}, nil
	}
	if paramNode.IsNil() {
		return &e.ParamSpec{}, nil
	}
	if paramNode.Kind != e.ListNode {
		return nil, fmt.Errorf("bad syntax: invalid parameter list")
	}
	spec := &e.ParamSpec{}
	for _, child := range paramNode.Children {
		spec.Fixed = append(spec.Fixed, child)
	}
	if paramNode.DottedTail != nil {
		spec.Variadic = paramNode.DottedTail
	}
	return spec, nil
}

func inheritLoc(child *e.Node, parent *e.Node) {
	if child != nil && child.Loc == nil && parent.Loc != nil {
		child.Loc = parent.Loc
	}
}

// --- Errors ---

type EvaluationError struct {
	msg   string
	Loc   e.CodeLocation
	cause error
}

func NewEvaluationError(msg string, loc e.CodeLocation) EvaluationError {
	return EvaluationError{msg: msg, Loc: loc}
}

func (err EvaluationError) Error() string {
	if err.Loc != nil {
		msg := fmt.Sprintf("%s: %s", err.Loc.String(), err.msg)
		if ctx := err.Loc.SourceContext(); ctx != "" {
			msg += "\n\n" + ctx
		}
		return msg
	}
	return err.msg
}

func (err EvaluationError) Unwrap() error {
	return err.cause
}

func (ev *Evaluator) pushContinuation(cont continuation) {
	*ev.conts = append(*ev.conts, cont)
}

func (ev *Evaluator) popContinuation() continuation {
	next := (*ev.conts)[len(*ev.conts)-1]
	*ev.conts = (*ev.conts)[:len(*ev.conts)-1]
	return next
}

func (ev *Evaluator) stepThroughContinuationsWithContext(ctx context.Context) (*e.Node, error) {
	var ret *e.Node = e.Nil
	var err error

	for len(*ev.conts) > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		next := ev.popContinuation()
		ret, err = next(ret, ev)

		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}
