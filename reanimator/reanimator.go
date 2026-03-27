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

	ev "github.com/archevel/ghoul/consume"
	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
	"github.com/archevel/ghoul/macromancy"
	"github.com/archevel/ghoul/tome"
)

type Reanimator struct {
	scopes      *macroScope
	evalEnv     *ev.Environment
	evaluator   *ev.Evaluator
	markCounter *uint64
	log         engraving.Logger
}

// generalSyntaxTransformer holds a user-defined lambda that acts as a
// macro transformer. The lambda is invoked through the evaluator during
// expansion so the transformer body can use the full language.
type generalSyntaxTransformer struct {
	fun ev.Function
}

// macroScope holds macro bindings with parent-chain scoping for inner
// define-syntax forms (e.g., inside lambda or begin blocks).
type macroScope struct {
	bindings map[e.Identifier]macroBinding
	parent   *macroScope
}

type macroBinding struct {
	syntaxTransformer  *macromancy.SyntaxTransformer
	generalTransformer *generalSyntaxTransformer
}

func newMacroScope(parent *macroScope) *macroScope {
	return &macroScope{
		bindings: make(map[e.Identifier]macroBinding),
		parent:   parent,
	}
}

func (s *macroScope) lookup(name e.Identifier) (macroBinding, bool) {
	for scope := s; scope != nil; scope = scope.parent {
		if b, ok := scope.bindings[name]; ok {
			return b, true
		}
	}
	return macroBinding{}, false
}

func (s *macroScope) define(name e.Identifier, b macroBinding) {
	s.bindings[name] = b
}

// New creates an Reanimator with its own evaluation environment for running
// general transformer bodies. The mark counter is shared with the evaluator
// that will process the expanded code.
func New(logger engraving.Logger, markCounter *uint64) *Reanimator {
	env := ev.NewEnvironment()
	tome.RegisterAll(env)
	evaluator := ev.NewWithMarkCounter(logger, env, markCounter)
	return &Reanimator{
		scopes:      newMacroScope(nil),
		evalEnv:     env,
		evaluator:   evaluator,
		markCounter: markCounter,
		log:         logger,
	}
}

func (exp *Reanimator) freshMark() macromancy.Mark {
	return atomic.AddUint64(exp.markCounter, 1)
}

// ExpandAll walks a top-level expression list and expands all macros.
// Returns a new expression list with no define-syntax forms or macro calls.
// Preserves source positions from the original parsed pairs.
func (exp *Reanimator) ExpandAll(exprs e.List) (e.List, error) {
	var results []e.Expr
	var resultPairs []*e.Pair // original pairs for source position preservation
	current := exprs
	for current != e.NIL {
		originalPair, _ := current.(*e.Pair)
		expr := current.First()
		expanded, err := exp.expandExpr(expr)
		if err != nil {
			return e.NIL, err
		}
		if expanded != nil {
			results = append(results, expanded)
			resultPairs = append(resultPairs, originalPair)
		}
		next, ok := current.Tail()
		if !ok {
			break
		}
		current = next
	}

	var result e.Expr = e.NIL
	for i := len(results) - 1; i >= 0; i-- {
		pair := e.Cons(results[i], result)
		if resultPairs[i] != nil && resultPairs[i].Loc != nil {
			pair.Loc = resultPairs[i].Loc
		}
		result = pair
	}
	if result == e.NIL {
		return e.NIL, nil
	}
	return result.(e.List), nil
}

// expandExpr returns the original expression unchanged when no macros are
// present, preserving source position information.
func (exp *Reanimator) expandExpr(expr e.Expr) (e.Expr, error) {
	list, isList := expr.(e.List)
	if !isList || list == e.NIL {
		return expr, nil
	}

	head := list.First()
	headId := identName(head)

	// define-syntax: register the macro, produce nothing in output
	if headId == "define-syntax" {
		return exp.processDefineSyntax(list)
	}

	// Known macro call: expand and re-process
	if headId != "" {
		if binding, found := exp.scopes.lookup(e.Identifier(headId)); found {
			expanded, err := exp.expandMacroCall(binding, list)
			if err != nil {
				return nil, err
			}
			return exp.expandExpr(expanded)
		}
	}

	// Check if any sub-expression needs expansion. If not, return
	// the original expression unchanged to preserve source positions.
	if !exp.containsMacroCall(list) {
		return expr, nil
	}

	switch headId {
	case "quote":
		return expr, nil
	case "require":
		return expr, nil
	case "lambda":
		return exp.expandLambda(list)
	case "begin":
		return exp.expandBegin(list)
	case "cond":
		return exp.expandCond(list)
	case "define":
		return exp.expandDefine(list)
	case "set!":
		return exp.expandSetBang(list)
	default:
		return exp.expandCall(list)
	}
}

// containsMacroCall checks whether any sub-expression in the list is a
// macro call or define-syntax. This allows the reanimator to skip unchanged
// expressions and preserve their source positions.
func (exp *Reanimator) containsMacroCall(list e.List) bool {
	current := list
	for current != e.NIL {
		elem := current.First()
		if subList, ok := elem.(e.List); ok && subList != e.NIL {
			subHead := identName(subList.First())
			if subHead == "define-syntax" {
				return true
			}
			if subHead != "" {
				if _, found := exp.scopes.lookup(e.Identifier(subHead)); found {
					return true
				}
			}
			if exp.containsMacroCall(subList) {
				return true
			}
		}
		next, ok := current.Tail()
		if !ok {
			break
		}
		current = next
	}
	return false
}

// processDefineSyntax handles (define-syntax name transformer).
// Returns nil to strip the form from the output.
func (exp *Reanimator) processDefineSyntax(form e.List) (e.Expr, error) {
	rest, ok := form.Tail()
	if !ok || rest == e.NIL {
		return nil, fmt.Errorf("bad syntax: define-syntax requires name and transformer")
	}

	name, nameOk := rest.First().(e.Identifier)
	if !nameOk {
		return nil, fmt.Errorf("bad syntax: define-syntax name must be an identifier")
	}

	transformerDef, transformerOk := rest.Tail()
	if !transformerOk || transformerDef == e.NIL {
		return nil, fmt.Errorf("bad syntax: define-syntax requires a transformer")
	}

	transformerExpr := transformerDef.First()
	transformerList, isList := transformerExpr.(e.List)
	if !isList {
		return nil, fmt.Errorf("bad syntax: transformer must be a form")
	}

	// syntax-rules: build transformer directly (no evaluation needed)
	if e.Identifier("syntax-rules").Equiv(transformerList.First()) {
		definitionBindings := exp.evalEnv.BoundIdentifierNames()
		st, err := macromancy.BuildSyntaxRulesTransformer(name, transformerList, definitionBindings)
		if err != nil {
			return nil, fmt.Errorf("bad syntax: %s", err)
		}
		exp.scopes.define(name, macroBinding{syntaxTransformer: &st})
		return nil, nil
	}

	// General transformer: first expand macro calls within the transformer
	// expression (e.g., the lambda body might call syntax-case), then
	// evaluate the expanded expression to get a Function.
	expandedTransformer, err := exp.expandExpr(transformerExpr)
	if err != nil {
		return nil, fmt.Errorf("define-syntax: failed to expand transformer: %w", err)
	}
	result, err := exp.evaluator.Evaluate(e.Cons(expandedTransformer, e.NIL))
	if err != nil {
		return nil, fmt.Errorf("define-syntax: failed to evaluate transformer: %w", err)
	}
	fun, isFun := result.(ev.Function)
	if !isFun {
		return nil, fmt.Errorf("bad syntax: transformer must be a procedure")
	}
	gst := &generalSyntaxTransformer{fun: fun}
	exp.scopes.define(name, macroBinding{generalTransformer: gst})
	return nil, nil
}

func (exp *Reanimator) expandMacroCall(binding macroBinding, callable e.List) (e.Expr, error) {
	if binding.syntaxTransformer != nil {
		mark := exp.freshMark()
		expanded, err := binding.syntaxTransformer.Transform(callable, mark)
		if err != nil {
			return nil, err
		}
		setMacroLocation(expanded, callable)
		return expanded, nil
	}

	if binding.generalTransformer != nil {
		mark := exp.freshMark()
		wrapped := macromancy.WrapExpr(callable, macromancy.NewMarkSet())
		markedInput := macromancy.ApplyMark(wrapped, mark)

		// Invoke the transformer via EvalSubExpression. The Function value
		// is embedded directly in the call expression (it self-evaluates),
		// and the marked input is quoted so it's passed as data.
		quotedInput := &e.Quote{Quoted: markedInput}
		callExpr := e.Cons(binding.generalTransformer.fun, e.Cons(quotedInput, e.NIL))
		result, err := exp.evaluator.EvalSubExpression(callExpr)
		if err != nil {
			return nil, err
		}

		marked := macromancy.ApplyMark(result, mark)
		resolved := macromancy.ResolveExpr(marked)
		setMacroLocation(resolved, callable)
		return resolved, nil
	}

	return nil, fmt.Errorf("internal error: macro binding has no transformer")
}

// --- Recursive expansion into sub-expressions ---

func (exp *Reanimator) expandLambda(form e.List) (e.Expr, error) {
	// (lambda params body ...)
	rest, ok := form.Tail()
	if !ok || rest == e.NIL {
		return form, nil
	}
	params := rest.First()
	body, ok := rest.Tail()
	if !ok {
		return form, nil
	}

	// Push a new macro scope for inner define-syntax
	saved := exp.scopes
	exp.scopes = newMacroScope(saved)
	defer func() { exp.scopes = saved }()

	expandedBody, err := exp.expandSequence(body)
	if err != nil {
		return nil, err
	}

	return rebuildList(form.First(), e.Cons(params, expandedBody)), nil
}

func (exp *Reanimator) expandBegin(form e.List) (e.Expr, error) {
	// (begin expr ...)
	rest, ok := form.Tail()
	if !ok {
		return form, nil
	}
	expandedBody, err := exp.expandSequence(rest)
	if err != nil {
		return nil, err
	}
	return rebuildList(form.First(), expandedBody), nil
}

func (exp *Reanimator) expandCond(form e.List) (e.Expr, error) {
	// (cond (pred consequent ...) ...)
	rest, ok := form.Tail()
	if !ok {
		return form, nil
	}
	var clauses []e.Expr
	current := rest
	for current != e.NIL {
		clause := current.First()
		clauseList, isList := clause.(e.List)
		if isList && clauseList != e.NIL {
			expanded, err := exp.expandEachInList(clauseList)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, expanded)
		} else {
			clauses = append(clauses, clause)
		}
		next, ok := current.Tail()
		if !ok {
			break
		}
		current = next
	}
	return rebuildList(form.First(), listFromSlice(clauses)), nil
}

func (exp *Reanimator) expandDefine(form e.List) (e.Expr, error) {
	// (define name value)
	rest, ok := form.Tail()
	if !ok || rest == e.NIL {
		return form, nil
	}
	name := rest.First()
	valueExpr, ok := rest.Tail()
	if !ok || valueExpr == e.NIL {
		return form, nil
	}
	expandedVal, err := exp.expandExpr(valueExpr.First())
	if err != nil {
		return nil, err
	}
	return rebuildList(form.First(), e.Cons(name, e.Cons(expandedVal, e.NIL))), nil
}

func (exp *Reanimator) expandSetBang(form e.List) (e.Expr, error) {
	// (set! name value) — same structure as define
	return exp.expandDefine(form)
}

func (exp *Reanimator) expandCall(form e.List) (e.Expr, error) {
	// (f arg1 arg2 ...) — expand each sub-expression
	return exp.expandEachInList(form)
}

// expandSequence expands a sequence of expressions (as in begin or lambda body).
// define-syntax forms are processed and stripped from the output.
func (exp *Reanimator) expandSequence(exprs e.List) (e.List, error) {
	var results []e.Expr
	current := exprs
	for current != e.NIL {
		expr := current.First()
		expanded, err := exp.expandExpr(expr)
		if err != nil {
			return e.NIL, err
		}
		if expanded != nil {
			results = append(results, expanded)
		}
		next, ok := current.Tail()
		if !ok {
			break
		}
		current = next
	}
	return listFromSlice(results), nil
}

func (exp *Reanimator) expandEachInList(list e.List) (e.Expr, error) {
	var elems []e.Expr
	current := list
	for current != e.NIL {
		expr := current.First()
		expanded, err := exp.expandExpr(expr)
		if err != nil {
			return nil, err
		}
		if expanded != nil {
			elems = append(elems, expanded)
		}
		next, ok := current.Tail()
		if !ok {
			// Improper list — preserve the dotted tail
			expandedTail, err := exp.expandExpr(current.Second())
			if err != nil {
				return nil, err
			}
			var result e.Expr = expandedTail
			for i := len(elems) - 1; i >= 0; i-- {
				result = e.Cons(elems[i], result)
			}
			return result, nil
		}
		current = next
	}
	return listFromSlice(elems), nil
}

// --- Helpers ---

func identName(expr e.Expr) string {
	switch v := expr.(type) {
	case e.Identifier:
		return string(v)
	case e.ScopedIdentifier:
		return string(v.Name)
	default:
		return ""
	}
}

func listFromSlice(exprs []e.Expr) e.List {
	if len(exprs) == 0 {
		return e.NIL
	}
	var result e.Expr = e.NIL
	for i := len(exprs) - 1; i >= 0; i-- {
		result = e.Cons(exprs[i], result)
	}
	return result.(e.List)
}

func rebuildList(head e.Expr, tail e.Expr) e.Expr {
	return e.Cons(head, tail)
}

// setMacroLocation stamps expanded code with the macro call site's location.
func setMacroLocation(expanded e.Expr, callSite e.List) {
	expandedPair, ok := expanded.(*e.Pair)
	if !ok {
		return
	}
	callPair, ok := callSite.(*e.Pair)
	if !ok {
		return
	}
	if callPair.Loc == nil {
		return
	}

	macroName := identName(callPair.H)
	loc := &e.MacroExpansionLocation{MacroName: macroName, CallSite: callPair.Loc}
	setLocationRecursive(expandedPair, loc)
}

func setLocationRecursive(pair *e.Pair, loc e.CodeLocation) {
	if pair.Loc == nil {
		pair.Loc = loc
	}
	if child, ok := pair.H.(*e.Pair); ok {
		setLocationRecursive(child, loc)
	}
	if child, ok := pair.T.(*e.Pair); ok {
		setLocationRecursive(child, loc)
	}
}
