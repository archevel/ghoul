// Package reanimator brings macromancy macros to life by expanding them
// into living code. It runs as a separate phase between parsing
// and evaluation — after reanimation, the expression tree contains no
// define-syntax forms and no macro calls, only core forms (lambda, define,
// set!, cond, begin, quote, require) and function calls.
//
// The reanimator maintains its own macro environment and uses a sub-evaluator
// (with stdlib registered) to execute general transformer bodies during
// expansion.
package reanimator

import (
	"fmt"
	"sync/atomic"

	"github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
	"github.com/archevel/ghoul/engraving"
	"github.com/archevel/ghoul/macromancy"
	"github.com/archevel/ghoul/tome"
)

type Reanimator struct {
	nodeScopes  *macroScope
	evalEnv     *ev.Environment
	evaluator   *ev.Evaluator
	markCounter *uint64
	log         engraving.Logger
}

// New creates a Reanimator with its own evaluation environment for running
// general transformer bodies. The mark counter is shared with the evaluator
// that will process the expanded code.
func New(logger engraving.Logger, markCounter *uint64) *Reanimator {
	env := ev.NewEnvironment()
	tome.RegisterAll(env)
	evaluator := ev.NewWithMarkCounter(logger, env, markCounter)
	return &Reanimator{
		evalEnv:     env,
		evaluator:   evaluator,
		markCounter: markCounter,
		log:         logger,
	}
}

func (exp *Reanimator) freshMark() macromancy.Mark {
	return atomic.AddUint64(exp.markCounter, 1)
}

// macroScope holds macro bindings keyed by string name.
type macroScope struct {
	bindings map[string]macroBinding
	parent   *macroScope
}

// generalTransformer holds a FuncNode that acts as a macro transformer.
type generalTransformer struct {
	funcNode *bones.Node
}

type macroBinding struct {
	syntaxTransformer  *macromancy.SyntaxTransformer
	generalTransformer *generalTransformer
}

func newMacroScope(parent *macroScope) *macroScope {
	return &macroScope{bindings: map[string]macroBinding{}, parent: parent}
}

func (s *macroScope) lookup(name string) (macroBinding, bool) {
	for scope := s; scope != nil; scope = scope.parent {
		if b, ok := scope.bindings[name]; ok {
			return b, true
		}
	}
	return macroBinding{}, false
}

func (s *macroScope) define(name string, b macroBinding) {
	s.bindings[name] = b
}

// ReanimateNodes expands all macros in a Node tree and translates the
// result into semantic nodes. This is the pipeline entry point.
func (exp *Reanimator) ReanimateNodes(topLevel *bones.Node) ([]*bones.Node, error) {
	if topLevel == nil || topLevel.IsNil() {
		return nil, nil
	}

	// Initialize the persistent node scope on first use
	if exp.nodeScopes == nil {
		exp.nodeScopes = newMacroScope(nil)
	}

	var results []*bones.Node
	for _, child := range topLevel.Children {
		expanded, err := exp.expandNode(child, exp.nodeScopes)
		if err != nil {
			return nil, err
		}
		if expanded != nil {
			results = append(results, expanded)
		}
	}

	// Translate ListNodes to semantic nodes (CallNode, LambdaNode, etc.)
	var translated []*bones.Node
	for _, node := range results {
		t, err := translateNode(node)
		if err != nil {
			return nil, err
		}
		// Propagate location from parent
		if t.Loc == nil && node.Loc != nil {
			t.Loc = node.Loc
		}
		translated = append(translated, t)
	}
	return translated, nil
}

func (exp *Reanimator) expandNode(node *bones.Node, scope *macroScope) (*bones.Node, error) {
	if node == nil || node.IsNil() {
		return node, nil
	}
	if node.Kind != bones.ListNode || len(node.Children) == 0 {
		return node, nil
	}

	head := node.Children[0]
	headName := head.IdentName()

	// define-syntax: register macro, strip from output
	if headName == "define-syntax" {
		return exp.processDefineSyntax(node, scope)
	}

	// Known macro call: expand and re-process
	if headName != "" {
		if binding, found := scope.lookup(headName); found {
			expanded, err := exp.expandMacroCall(binding, node, scope)
			if err != nil {
				return nil, err
			}
			return exp.expandNode(expanded, scope)
		}
	}

	// No macro calls in subtree? Return unchanged
	if !exp.containsMacroCall(node, scope) {
		return node, nil
	}

	switch headName {
	case "quote", "require":
		return node, nil
	case "lambda":
		return exp.expandLambda(node, scope)
	case "begin":
		return exp.expandBegin(node, scope)
	case "cond":
		return exp.expandCond(node, scope)
	case "define":
		return exp.expandDefine(node, scope)
	case "set!":
		return exp.expandDefine(node, scope)
	default:
		return exp.expandEach(node, scope)
	}
}

func (exp *Reanimator) containsMacroCall(node *bones.Node, scope *macroScope) bool {
	if node.Kind != bones.ListNode {
		return false
	}
	for _, child := range node.Children {
		if child.Kind == bones.ListNode && len(child.Children) > 0 {
			name := child.Children[0].IdentName()
			if name == "define-syntax" {
				return true
			}
			if name != "" {
				if _, found := scope.lookup(name); found {
					return true
				}
			}
			if exp.containsMacroCall(child, scope) {
				return true
			}
		}
	}
	return false
}

func (exp *Reanimator) processDefineSyntax(node *bones.Node, scope *macroScope) (*bones.Node, error) {
	if len(node.Children) < 3 {
		return nil, fmt.Errorf("bad syntax: define-syntax requires name and transformer")
	}

	nameNode := node.Children[1]
	name := nameNode.IdentName()
	if name == "" {
		return nil, fmt.Errorf("bad syntax: define-syntax name must be an identifier")
	}

	transformerNode := node.Children[2]
	if transformerNode.Kind != bones.ListNode || len(transformerNode.Children) == 0 {
		return nil, fmt.Errorf("bad syntax: transformer must be a form")
	}

	// syntax-rules: build Node-based transformer directly
	if transformerNode.Children[0].IdentName() == "syntax-rules" {
		defBindings := exp.boundIdentifierNames()
		st, err := macromancy.BuildSyntaxRulesTransformer(name, transformerNode, defBindings)
		if err != nil {
			return nil, fmt.Errorf("bad syntax: %s", err)
		}
		scope.define(name, macroBinding{syntaxTransformer: &st})
		return nil, nil
	}

	// General transformer: expand, translate, evaluate to get a Function,
	// then store it for later invocation during macro calls.
	expandedNode, err := exp.expandNode(transformerNode, scope)
	if err != nil {
		return nil, fmt.Errorf("define-syntax: failed to expand transformer: %w", err)
	}
	translated, err := translateNode(expandedNode)
	if err != nil {
		return nil, fmt.Errorf("define-syntax: failed to translate transformer: %w", err)
	}
	resultNode, err := exp.evaluator.ConsumeNodes([]*bones.Node{translated})
	if err != nil {
		return nil, fmt.Errorf("define-syntax: failed to evaluate transformer: %w", err)
	}
	if resultNode.Kind != bones.FunctionNode || resultNode.FuncVal == nil {
		return nil, fmt.Errorf("bad syntax: transformer must be a procedure")
	}
	scope.define(name, macroBinding{generalTransformer: &generalTransformer{funcNode: resultNode}})
	return nil, nil
}

func (exp *Reanimator) expandMacroCall(binding macroBinding, node *bones.Node, scope *macroScope) (*bones.Node, error) {
	if binding.syntaxTransformer != nil {
		mark := exp.freshMark()
		expanded, err := binding.syntaxTransformer.Transform(node, mark)
		if err != nil {
			return nil, err
		}
		setMacroLocation(expanded, node)
		return expanded, nil
	}

	if binding.generalTransformer != nil {
		mark := exp.freshMark()

		// Node-based hygiene: wrap, mark, invoke, mark again, resolve
		wrapped := macromancy.WrapSyntax(node, macromancy.NewMarkSet())
		markedInput := macromancy.ApplyMark(wrapped, mark)

		// Build a call node: (transformer-fn (quote markedInput))
		quotedInput := bones.QuoteNodeVal(markedInput)
		callNode := &bones.Node{
			Kind:     bones.CallNode,
			Children: []*bones.Node{binding.generalTransformer.funcNode, quotedInput},
		}
		resultNode, err := exp.evaluator.EvalSubExpression(callNode)
		if err != nil {
			return nil, err
		}

		// Apply mark again (toggle cancels input marks) and resolve
		marked := macromancy.ApplyMark(resultNode, mark)
		resolved := macromancy.ResolveSyntax(marked)
		setMacroLocation(resolved, node)
		return resolved, nil
	}

	return nil, fmt.Errorf("internal error: macro binding has no transformer")
}

func (exp *Reanimator) expandLambda(node *bones.Node, scope *macroScope) (*bones.Node, error) {
	if len(node.Children) < 3 {
		return node, nil
	}
	saved := scope
	scope = newMacroScope(saved)
	defer func() { scope = saved }()

	expandedBody, err := exp.expandSequence(node.Children[2:], scope)
	if err != nil {
		return nil, err
	}

	children := make([]*bones.Node, 0, 2+len(expandedBody))
	children = append(children, node.Children[0]) // lambda keyword
	children = append(children, node.Children[1]) // params
	children = append(children, expandedBody...)
	return &bones.Node{Kind: bones.ListNode, Children: children, Loc: node.Loc}, nil
}

func (exp *Reanimator) expandBegin(node *bones.Node, scope *macroScope) (*bones.Node, error) {
	if len(node.Children) < 2 {
		return node, nil
	}
	expandedBody, err := exp.expandSequence(node.Children[1:], scope)
	if err != nil {
		return nil, err
	}
	children := make([]*bones.Node, 0, 1+len(expandedBody))
	children = append(children, node.Children[0]) // begin keyword
	children = append(children, expandedBody...)
	return &bones.Node{Kind: bones.ListNode, Children: children, Loc: node.Loc}, nil
}

func (exp *Reanimator) expandCond(node *bones.Node, scope *macroScope) (*bones.Node, error) {
	if len(node.Children) < 2 {
		return node, nil
	}
	children := []*bones.Node{node.Children[0]} // cond keyword
	for _, clause := range node.Children[1:] {
		if clause.Kind == bones.ListNode && len(clause.Children) > 0 {
			expanded, err := exp.expandEach(clause, scope)
			if err != nil {
				return nil, err
			}
			children = append(children, expanded)
		} else {
			children = append(children, clause)
		}
	}
	return &bones.Node{Kind: bones.ListNode, Children: children, Loc: node.Loc}, nil
}

func (exp *Reanimator) expandDefine(node *bones.Node, scope *macroScope) (*bones.Node, error) {
	if len(node.Children) < 3 {
		return node, nil
	}
	expandedVal, err := exp.expandNode(node.Children[2], scope)
	if err != nil {
		return nil, err
	}
	return &bones.Node{
		Kind:     bones.ListNode,
		Children: []*bones.Node{node.Children[0], node.Children[1], expandedVal},
		Loc:      node.Loc,
	}, nil
}

func (exp *Reanimator) expandSequence(nodes []*bones.Node, scope *macroScope) ([]*bones.Node, error) {
	var results []*bones.Node
	for _, child := range nodes {
		expanded, err := exp.expandNode(child, scope)
		if err != nil {
			return nil, err
		}
		if expanded != nil {
			results = append(results, expanded)
		}
	}
	return results, nil
}

func (exp *Reanimator) expandEach(node *bones.Node, scope *macroScope) (*bones.Node, error) {
	if node.Kind != bones.ListNode {
		return node, nil
	}
	children := make([]*bones.Node, len(node.Children))
	for i, child := range node.Children {
		expanded, err := exp.expandNode(child, scope)
		if err != nil {
			return nil, err
		}
		children[i] = expanded
	}
	return &bones.Node{Kind: bones.ListNode, Children: children, Loc: node.Loc}, nil
}

func (exp *Reanimator) boundIdentifierNames() map[string]bool {
	return exp.evalEnv.BoundIdentifierNames()
}

// --- Node-based translate ---

func translateNode(node *bones.Node) (*bones.Node, error) {
	if node == nil || node.IsNil() {
		return node, nil
	}

	// Non-list nodes pass through unchanged
	if node.Kind != bones.ListNode {
		return node, nil
	}

	// Empty list
	if len(node.Children) == 0 {
		return node, nil
	}

	headName := node.Children[0].IdentName()
	switch headName {
	case "quote":
		return translateQuote(node)
	case "define":
		return translateDefineOrSet(node, bones.DefineNode)
	case "set!":
		return translateDefineOrSet(node, bones.SetNode)
	case "lambda":
		return translateLambda(node)
	case "cond":
		return translateCond(node)
	case "begin":
		return translateBegin(node)
	case "require":
		return translateRequire(node)
	default:
		return translateCall(node)
	}
}

func translateQuote(node *bones.Node) (*bones.Node, error) {
	if len(node.Children) < 2 {
		return nil, fmt.Errorf("bad syntax: quote requires an argument")
	}
	return &bones.Node{Kind: bones.QuoteNode, Quoted: node.Children[1], Loc: node.Loc}, nil
}

func translateDefineOrSet(node *bones.Node, kind bones.NodeKind) (*bones.Node, error) {
	if len(node.Children) < 3 {
		return nil, fmt.Errorf("bad syntax: missing value in binding")
	}
	nameNode := node.Children[1]
	valueNode, err := translateNode(node.Children[2])
	if err != nil {
		return nil, err
	}
	inheritLoc(nameNode, node)
	inheritLoc(valueNode, node)
	return &bones.Node{
		Kind:     kind,
		Loc:      node.Loc,
		Children: []*bones.Node{nameNode, valueNode},
	}, nil
}

func translateLambda(node *bones.Node) (*bones.Node, error) {
	if len(node.Children) < 3 {
		return nil, fmt.Errorf("bad syntax: lambda requires parameters and body")
	}
	params, err := translateParams(node.Children[1])
	if err != nil {
		return nil, err
	}
	var bodyNodes []*bones.Node
	for _, child := range node.Children[2:] {
		t, err := translateNode(child)
		if err != nil {
			return nil, err
		}
		inheritLoc(t, node)
		bodyNodes = append(bodyNodes, t)
	}
	return &bones.Node{
		Kind:     bones.LambdaNode,
		Loc:      node.Loc,
		Params:   params,
		Children: bodyNodes,
	}, nil
}

func translateParams(paramNode *bones.Node) (*bones.ParamSpec, error) {
	// Single identifier: all-variadic
	if paramNode.Kind == bones.IdentifierNode {
		return &bones.ParamSpec{Variadic: paramNode}, nil
	}
	if paramNode.Kind != bones.ListNode {
		return nil, fmt.Errorf("bad syntax: invalid parameter list")
	}
	spec := &bones.ParamSpec{}
	for _, child := range paramNode.Children {
		spec.Fixed = append(spec.Fixed, child)
	}
	if paramNode.DottedTail != nil {
		spec.Variadic = paramNode.DottedTail
	}
	return spec, nil
}

func translateCond(node *bones.Node) (*bones.Node, error) {
	var clauses []*bones.CondClause
	for _, clauseNode := range node.Children[1:] {
		if clauseNode.Kind != bones.ListNode || len(clauseNode.Children) == 0 {
			return nil, fmt.Errorf("bad syntax: cond clause must be a list")
		}
		clause, err := translateCondClause(clauseNode)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause)
	}
	return &bones.Node{Kind: bones.CondNode, Loc: node.Loc, Clauses: clauses}, nil
}

func translateCondClause(clauseNode *bones.Node) (*bones.CondClause, error) {
	test := clauseNode.Children[0]
	isElse := test.IdentName() == "else"

	var bodyNodes []*bones.Node
	for _, child := range clauseNode.Children[1:] {
		t, err := translateNode(child)
		if err != nil {
			return nil, err
		}
		bodyNodes = append(bodyNodes, t)
	}

	result := &bones.CondClause{IsElse: isElse, Consequent: bodyNodes}
	if !isElse {
		testNode, err := translateNode(test)
		if err != nil {
			return nil, err
		}
		result.Test = testNode
	}
	return result, nil
}

func translateBegin(node *bones.Node) (*bones.Node, error) {
	var bodyNodes []*bones.Node
	for _, child := range node.Children[1:] {
		t, err := translateNode(child)
		if err != nil {
			return nil, err
		}
		inheritLoc(t, node)
		bodyNodes = append(bodyNodes, t)
	}
	return &bones.Node{Kind: bones.BeginNode, Loc: node.Loc, Children: bodyNodes}, nil
}

func translateRequire(node *bones.Node) (*bones.Node, error) {
	return &bones.Node{Kind: bones.RequireNode, Loc: node.Loc, RawArgs: node.Children[1:]}, nil
}

func translateCall(node *bones.Node) (*bones.Node, error) {
	children := make([]*bones.Node, len(node.Children))
	for i, child := range node.Children {
		t, err := translateNode(child)
		if err != nil {
			return nil, err
		}
		inheritLoc(t, node)
		children[i] = t
	}
	return &bones.Node{Kind: bones.CallNode, Loc: node.Loc, Children: children}, nil
}

func inheritLoc(child *bones.Node, parent *bones.Node) {
	if child.Loc == nil && parent.Loc != nil {
		child.Loc = parent.Loc
	}
}

func setMacroLocation(expanded *bones.Node, callSite *bones.Node) {
	if expanded == nil || callSite == nil || callSite.Loc == nil {
		return
	}
	macroName := ""
	if len(callSite.Children) > 0 {
		macroName = callSite.Children[0].IdentName()
	}
	loc := &bones.MacroExpansionLocation{MacroName: macroName, CallSite: callSite.Loc}
	setLocationRecursive(expanded, loc)
}

func setLocationRecursive(node *bones.Node, loc bones.CodeLocation) {
	if node.Loc == nil {
		node.Loc = loc
	}
	if node.Kind == bones.ListNode {
		for _, child := range node.Children {
			setLocationRecursive(child, loc)
		}
	}
}
