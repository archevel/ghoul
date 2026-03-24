package evaluator

import (
	"context"
	"fmt"
	"sync/atomic"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
	"github.com/archevel/ghoul/macromancy"
)

const COND_SPECIAL_FORM = e.Identifier("cond")
const ELSE_SPECIAL_FORM = e.Identifier("else")
const BEGIN_SPECIAL_FORM = e.Identifier("begin")
const LAMBDA_SPECIAL_FORM = e.Identifier("lambda")
const DEFINE_SPECIAL_FORM = e.Identifier("define")
const ASSIGNMENT_SPECIAL_FORM = e.Identifier("set!")
const DEFINE_SYNTAX_SPECIAL_FORM = e.Identifier("define-syntax")
const SYNTAX_RULES_FORM = e.Identifier("syntax-rules")

var markCounter uint64

func freshMark() macromancy.Mark {
	return atomic.AddUint64(&markCounter, 1)
}

// SyntaxTransformer is bound in the environment by define-syntax.
// When the evaluator sees a call whose head resolves to one, it
// expands the macro instead of evaluating arguments.
type SyntaxTransformer struct {
	Transform func(code e.List, mark macromancy.Mark) (e.Expr, error)
}

func (st SyntaxTransformer) Repr() string {
	return "#<syntax-transformer>"
}

func (st SyntaxTransformer) Equiv(other e.Expr) bool {
	return false
}

// GeneralSyntaxTransformer holds a user-defined lambda that acts as a
// macro transformer. Unlike SyntaxTransformer (which does pattern-based
// expansion directly), this invokes the lambda through the continuation
// stack so the transformer body can use the full language.
type GeneralSyntaxTransformer struct {
	Fun Function
}

func (gst GeneralSyntaxTransformer) Repr() string {
	return "#<general-syntax-transformer>"
}

func (gst GeneralSyntaxTransformer) Equiv(other e.Expr) bool {
	return false
}

type continuation func(arg e.Expr, ev *Evaluator) (e.Expr, error)
type contStack []continuation

func Evaluate(exprs e.Expr, env *environment) (res e.Expr, err error) {
	return EvaluateWithContext(context.Background(), exprs, env)
}

func EvaluateWithContext(ctx context.Context, exprs e.Expr, env *environment) (res e.Expr, err error) {
	evaluator := New(logging.StandardLogger, env)
	return evaluator.EvaluateWithContext(ctx, exprs)
}

func New(logger logging.Logger, env *environment) *Evaluator {
	return &Evaluator{log: logger, env: env}
}

type Evaluator struct {
	log   logging.Logger
	env   *environment
	conts *contStack
}

func (ev *Evaluator) Evaluate(exprs e.Expr) (e.Expr, error) {
	return ev.EvaluateWithContext(context.Background(), exprs)
}

func (ev *Evaluator) GetEnvironment() *environment {
	return ev.env
}

func (ev *Evaluator) EvaluateWithContext(ctx context.Context, exprs e.Expr) (e.Expr, error) {
	if exprs == e.NIL {
		return exprs, nil
	}
	listExpr := wrapNonList(exprs)
	ev.conts = &contStack{sexprSeqEvalContinuationFor(listExpr, false)}

	return ev.stepThroughContinuationsWithContext(ctx)
}

func (ev *Evaluator) stepThroughContinuationsWithContext(ctx context.Context) (e.Expr, error) {
	var ret e.Expr = e.NIL
	var err error

	ev.log.Trace("Starting to step through continuations")
	for len(*ev.conts) > 0 {
		// Check for context cancellation/timeout on each iteration
		select {
		case <-ctx.Done():
			ev.log.Trace("Evaluation canceled due to context")
			return nil, ctx.Err()
		default:
		}

		next := ev.popContinuation()
		ret, err = next(ret, ev)

		if err != nil {
			ev.log.Trace("Continuation returned an error!")
			return nil, err
		}
	}

	ev.log.Trace("Nothing left to evaluate. Returning %s", ret)
	return ret, nil
}

func (ev *Evaluator) pushContinuation(cont continuation) {
	var conts *contStack = ev.conts
	*conts = append(*conts, cont)
}

func (ev *Evaluator) popContinuation() continuation {
	next := (*ev.conts)[len(*ev.conts)-1]
	*ev.conts = (*ev.conts)[:len(*ev.conts)-1]
	return next
}

func sexprSeqEvalContinuationFor(exprs e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		t, ok := exprs.Tail()
		if ok && t != e.NIL {
			ev.log.Trace("Pushing continuation for evaluating tail of expression sequence")
			ev.pushContinuation(sexprSeqEvalContinuationFor(t, maybeTailCall))
		} else if !ok {
			return nil, NewEvaluationError("Malformed expresion sequence", exprs)
		}
		head := exprs.Head()
		ev.log.Trace("Pushing continuation for evaluating head of expression sequence")
		ev.pushContinuation(sexprEvalContinuationFor(head, exprs, maybeTailCall && t == e.NIL))
		return e.NIL, nil
	}
}

func sexprEvalContinuationFor(expr e.Expr, parent e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		ev.log.Trace("Choosing evaluation continuation")
		ret, nextCont, err := chooseEvaluation(expr, parent, maybeTailCall)
		if nextCont != nil {
			ev.log.Trace("Pushing choice")
			ev.pushContinuation(nextCont)
		}

		return ret, err
	}
}

// specialFormName extracts the identifier name from h for special form
// matching, ignoring hygiene marks. Special forms are language primitives,
// not bindings, so a macro-introduced "begin" should still be recognized.
func specialFormName(h e.Expr) e.Identifier {
	switch v := h.(type) {
	case e.Identifier:
		return v
	case e.ScopedIdentifier:
		return v.Name
	default:
		return ""
	}
}

func chooseEvaluation(expr e.Expr, parent e.List, maybeTailCall bool) (ret e.Expr, nextCont continuation, err error) {
	switch v := expr.(type) {
	case *e.Quote:
		ret = v.Quoted
	case e.Identifier:
		nextCont = makeIdentificationLookupContinuationFor(v, parent)
		ret = e.NIL
	case e.ScopedIdentifier:
		nextCont = makeIdentificationLookupContinuationFor(v, parent)
		ret = e.NIL
	case e.List:
		h, t, isList := maybeSplitExpr(expr)
		if !isList {
			err = NewEvaluationError("Malformed expression", parent)
			return
		}

		ret = e.NIL
		switch specialFormName(h) {
		case DEFINE_SYNTAX_SPECIAL_FORM:
			nextCont = defineSyntaxContinuationFor(t)
		case DEFINE_SPECIAL_FORM:
			nextCont = defineContinuationFor(t, maybeTailCall)
		case LAMBDA_SPECIAL_FORM:
			nextCont = lambdaContinuationFor(t)
		case COND_SPECIAL_FORM:
			nextCont = conditionalContinuationFor(t, maybeTailCall)
		case ASSIGNMENT_SPECIAL_FORM:
			nextCont = assignmentContinuationFor(t, maybeTailCall)
		case BEGIN_SPECIAL_FORM:
			nextCont = sexprSeqEvalContinuationFor(t, maybeTailCall)
		default:
			nextCont = functionCallContinuationFor(v, maybeTailCall)
		}
	default:
		ret = expr
	}

	return
}

func makeIdentificationLookupContinuationFor(ident e.Expr, parent e.List) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		var env *environment = ev.env

		ev.log.Trace("Looking up identifier: %s", ident)
		resExpr, err := lookupIdentifier(ident, env)
		if err != nil {
			ev.log.Trace("Failed looking up identifier: %s", ident)
			err = NewEvaluationError(err.Error(), parent)
			return e.NIL, err
		}
		return resExpr, nil

	}
}
func assignmentContinuationFor(assignment e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		valueExpr, valOk := assignment.Tail()
		if !valOk {
			ev.log.Trace("Failed evaluate assignment: %s", assignment)
			return e.NIL, NewEvaluationError("Malformed assignment", assignment)
		}
		nilTail, nilOk := valueExpr.Tail()
		if valOk && nilOk && valueExpr != e.NIL && nilTail == e.NIL {
			ev.log.Trace("Pushing anonymous assigment func: %s", assignment)
			ev.pushContinuation(func(value e.Expr, ev *Evaluator) (e.Expr, error) {
				var env *environment = ev.env

				ret, err := assign(assignment.Head(), value, env)
				return ret, err
			})
			ev.pushContinuation(sexprEvalContinuationFor(valueExpr.Head(), valueExpr, maybeTailCall))
			return e.NIL, nil

		} else {
			ev.log.Trace("Tail part of assignment expession was malformed: %s", assignment)
			return e.NIL, NewEvaluationError("Malformed assignment", assignment)
		}
	}

}

func conditionalContinuationFor(conds e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		if conds == e.NIL {
			ev.log.Trace("No condition successfully matched so returning NIL")
			return e.NIL, nil
		}

		alternative, ok := headList(conds)
		if !ok {
			ev.log.Trace("Malformed alternative of cond list. Head should be list, but was %s", conds.Head())
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+conds.Head().Repr(), conds)
		}

		if alternative == e.NIL {
			ev.log.Trace("Malformed alternative of cond list. Alternative was NIL")
			return nil, NewEvaluationError("Bad syntax: Missing condition", conds)
		}

		consequent, ok := alternative.Tail()
		if !ok {
			ev.log.Trace("Malformed alternative of cond list. Alternative likely in invalid pair form: %s", alternative)
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+alternative.Repr(), alternative)
		}

		if consequent == e.NIL {
			ev.log.Trace("Malformed alternative of cond list. Alternative tail was NIL in: %s", alternative)
			return nil, NewEvaluationError("Bad syntax: Missing consequent", alternative)
		}

		predExpr := alternative.Head()
		if specialFormName(predExpr) == ELSE_SPECIAL_FORM {
			predExpr = e.Boolean(true)
		}

		nextPredOrConsequent := func(truthy e.Expr, ev *Evaluator) (e.Expr, error) {
			if isTruthy(truthy) {
				ev.log.Trace("Found truthy alternative pushing evaluation of consequent")
				ev.pushContinuation(sexprEvalContinuationFor(consequent.Head(), conds, maybeTailCall))
				return e.NIL, nil
			}

			tailConds, ok := conds.Tail()
			if !ok {
				ev.log.Trace("Malformed cond list. Tail was not a list in: %s", conds)
				return nil, NewEvaluationError("Bad syntax: Malformed cond, expected list not pair", conds)
			}

			ev.log.Trace("Trying next alternative in cond list")
			ev.pushContinuation(conditionalContinuationFor(tailConds, maybeTailCall))
			return e.NIL, nil
		}

		ev.log.Trace("Pushing evaluation of nextPredOrConsequent followed by evaluation of current alternative")
		ev.pushContinuation(nextPredOrConsequent)
		ev.pushContinuation(sexprEvalContinuationFor(predExpr, alternative, false))

		return e.NIL, nil
	}

}

func lambdaContinuationFor(lambda e.List) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		var env *environment = ev.env
		if body, ok := lambda.Tail(); ok {
			fun := func(args e.List, ev *Evaluator) (e.Expr, error) {
				ev.log.Trace("Pushing evaluation of lambda body and preparation of the lambdas scope")
				ev.pushContinuation(sexprSeqEvalContinuationFor(body, true))
				ev.pushContinuation(prepareScope(lambda.Head(), args, env))

				return e.NIL, nil
			}

			ev.log.Trace("Yielding Function expression value for lambda")
			return Function{&fun}, nil
		} else {
			ev.log.Trace("Lambda expression had malformed body: %s", lambda)
			return e.NIL, NewEvaluationError("Malformed lambda expression", lambda)
		}
	}
}

func prepareScope(paramExpr e.Expr, args e.List, definitionEnv *environment) continuation {

	return func(ignore e.Expr, ev *Evaluator) (e.Expr, error) {
		ev.log.Trace("Preparing new scope for function evaluation")
		newEnv := newEnvWithEmptyScope(definitionEnv)

		paramList, ok := paramExpr.(e.List)
		var variadicParam e.Expr = paramExpr

		ev.log.Trace("Binding function arguments %s to parameter list %s", args, paramList)
		for ok && paramList != e.NIL && args != e.NIL {
			arg := args.Head()
			args, _ = args.Tail()
			param := paramList.Head()
			pl, ok := paramList.Tail()
			if !ok {
				variadicParam = paramList.Second()
				paramList = e.NIL
			} else {
				paramList = pl
			}
			bindIdentifier(param, arg, newEnv)
		}

		if variadicId, ok := variadicParam.(e.Identifier); ok {
			ev.log.Trace("Binding remaining args %s to variadic parameter %s", args, variadicId)
			bindIdentifier(variadicId, args, newEnv)
		} else if args != e.NIL {
			ev.log.Trace("More arguments given than the function supports!")
			return e.NIL, NewEvaluationError("Arity mismatch: too many arguments", args)
		} else if paramList != e.NIL {
			ev.log.Trace("Not all parameters could be given a value!")
			return e.NIL, NewEvaluationError("Arity mismatch: too few arguments", args)
		}

		ev.log.Trace("Reassigning evaluator environment pointer to new environment")
		ev.env = newEnv
		return e.NIL, nil
	}
}

func functionCallContinuationFor(callable e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		ev.log.Trace("Evaluating function call")

		if callable == e.NIL {
			ev.log.Trace("NIL is not a function that can be called!")
			return e.NIL, NewEvaluationError("Missing procedure expression in: ()", callable)
		}

		// Macro calls must be intercepted before argument evaluation,
		// since macros receive unevaluated syntax, not values.
		if headVal, _ := resolveCallableHead(callable.Head(), ev.env); headVal != nil {
			if st, ok := headVal.(SyntaxTransformer); ok {
				ev.log.Trace("Expanding syntax transformer")
				mark := freshMark()
				expanded, err := st.Transform(callable, mark)
				if err != nil {
					return e.NIL, NewEvaluationError(err.Error(), callable)
				}
				setMacroLocation(expanded, callable)
				ev.pushContinuation(sexprEvalContinuationFor(expanded, callable, maybeTailCall))
				return e.NIL, nil
			}
			if gst, ok := headVal.(GeneralSyntaxTransformer); ok {
				ev.log.Trace("Expanding general syntax transformer")
				mark := freshMark()
				// Apply the mark to the input before passing it to the transformer.
				// After the transformer returns, we apply the same mark again.
				// Identifiers from the input get the mark twice (cancels via toggle),
				// while identifiers introduced by the transformer get it once (stays).
				wrapped := macromancy.WrapExpr(callable, macromancy.NewMarkSet())
				markedInput := macromancy.ApplyMark(wrapped, mark)

				ev.pushContinuation(func(result e.Expr, ev *Evaluator) (e.Expr, error) {
					marked := macromancy.ApplyMark(result, mark)
					resolved := macromancy.ResolveExpr(marked)
					setMacroLocation(resolved, callable)
					ev.pushContinuation(sexprEvalContinuationFor(resolved, callable, maybeTailCall))
					return e.NIL, nil
				})

				proc := gst.Fun.Fun
				res, err := (*proc)(e.Cons(markedInput, e.NIL), ev)
				if err != nil {
					return e.NIL, NewEvaluationError(err.Error(), callable)
				}
				return res, nil
			}
		}

		var callEnv *environment = ev.env

		if !maybeTailCall {
			ev.log.Trace("Pushing environment restoration to evaluate after function call since the call is not made in tail position")
			ev.pushContinuation(func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
				ev.env = callEnv
				return arg, nil
			})
		}

		var argList e.List = e.NIL
		collectArgs := func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
			argList = cons(arg, argList)
			return argList, nil
		}

		applyFunc := func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
			funExpr, ok := arg.(Function)
			if !ok {
				ev.log.Trace("Can not apply a non-function!")
				return e.NIL, NewEvaluationError("Not a procedure: "+arg.Repr(), callable)
			}
			proc := funExpr.Fun

			ev.log.Trace("Applying function with arguments collected")
			res, err := (*proc)(argList, ev)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return res, err
				}
				return res, NewEvaluationError(err.Error(), callable)
			}
			return res, nil
		}
		ev.log.Trace("Pushing function application and function resolution")
		ev.pushContinuation(applyFunc)
		resolveFunc := sexprEvalContinuationFor(callable.Head(), callable, false)
		ev.pushContinuation(resolveFunc)

		funcArgs, ok := callable.Tail()
		ev.log.Trace("Pushing collection and evaluation of function arguments")
		for ok && funcArgs != e.NIL {
			anArg := funcArgs.Head()
			ev.pushContinuation(collectArgs)
			ev.pushContinuation(sexprEvalContinuationFor(anArg, callable, false))
			funcArgs, ok = funcArgs.Tail()
			if !ok {
				ev.log.Trace("Function call is malformed in: %s", callable)
				return e.NIL, NewEvaluationError("Bad syntax in procedure application", funcArgs)
			}
		}
		return e.NIL, nil
	}
}

func defineContinuationFor(def e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		valueExpr, valOk := def.Tail()
		if !valOk {
			ev.log.Trace("Define has inproper fromat: %s", def)
			return nil, NewEvaluationError("Bad syntax: invalid binding format", def)
		}

		if valueExpr == e.NIL {
			ev.log.Trace("Define was not given a value to bind to identifier in: %s", def)
			return nil, NewEvaluationError("Bad syntax: missing value in binding", def)
		}
		if t, ok := valueExpr.Tail(); ok && t != e.NIL {
			ev.log.Trace("Define was given more than one argument after binding identifier: %s", def)
			return nil, NewEvaluationError("Bad syntax: multiple values in binding", def)
		}

		ev.log.Trace("Pushing binding of variable and evaluation of definition value")
		ev.pushContinuation(bindVar(def.Head()))
		ev.pushContinuation(sexprEvalContinuationFor(valueExpr.Head(), valueExpr, maybeTailCall))
		return e.NIL, nil
	}
}

func bindVar(expr e.Expr) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		var env *environment = ev.env
		ev.log.Trace("Binding identifier: %s to: %s", expr, arg)
		res, err := bindIdentifier(expr, arg, env)
		return res, err
	}
}

// setMacroLocation stamps expanded code with the macro call site's location
// so errors in expanded code point back to where the macro was used.
func setMacroLocation(expanded e.Expr, callSite e.List) {
	expandedPair, ok := expanded.(*e.Pair)
	if !ok {
		return
	}
	callPair, ok := callSite.(*e.Pair)
	if !ok {
		return
	}

	var callLoc e.CodeLocation
	if callPair.Loc != nil {
		callLoc = callPair.Loc
	} else {
		return
	}

	macroName := ""
	switch h := callPair.H.(type) {
	case e.Identifier:
		macroName = string(h)
	case e.ScopedIdentifier:
		macroName = string(h.Name)
	}

	loc := &e.MacroExpansionLocation{MacroName: macroName, CallSite: callLoc}
	setLocationRecursive(expandedPair, loc)
}

// setLocationRecursive sets the location on all Pairs in a tree
// that don't already have one.
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

func resolveCallableHead(head e.Expr, env *environment) (e.Expr, string) {
	switch h := head.(type) {
	case e.Identifier:
		if val, err := lookupIdentifier(h, env); err == nil {
			return val, string(h)
		}
	case e.ScopedIdentifier:
		if val, err := lookupIdentifier(h, env); err == nil {
			return val, string(h.Name)
		}
	}
	return nil, ""
}

func defineSyntaxContinuationFor(def e.List) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		if def == e.NIL {
			return e.NIL, NewEvaluationError("Bad syntax: define-syntax requires name and transformer", def)
		}

		name, nameOk := def.First().(e.Identifier)
		if !nameOk {
			return e.NIL, NewEvaluationError("Bad syntax: define-syntax name must be an identifier", def)
		}

		transformerDef, transformerOk := def.Tail()
		if !transformerOk || transformerDef == e.NIL {
			return e.NIL, NewEvaluationError("Bad syntax: define-syntax requires a transformer", def)
		}

		transformerExpr := transformerDef.First()
		transformerList, isList := transformerExpr.(e.List)
		if !isList {
			return e.NIL, NewEvaluationError("Bad syntax: transformer must be a form", def)
		}

		if SYNTAX_RULES_FORM.Equiv(transformerList.First()) {
			transformer, err := buildSyntaxRulesTransformer(name, transformerList, ev.env)
			if err != nil {
				return e.NIL, NewEvaluationError(fmt.Sprintf("Bad syntax: %s", err), def)
			}

			bindIdentifier(name, transformer, ev.env)
			return e.NIL, nil
		}

		ev.pushContinuation(func(transformerVal e.Expr, ev *Evaluator) (e.Expr, error) {
			fun, isFun := transformerVal.(Function)
			if !isFun {
				return e.NIL, NewEvaluationError("Bad syntax: transformer must be a procedure", def)
			}
			st := GeneralSyntaxTransformer{
				Fun: fun,
			}
			bindIdentifier(name, st, ev.env)
			return e.NIL, nil
		})
		ev.pushContinuation(sexprEvalContinuationFor(transformerExpr, transformerDef, false))
		return e.NIL, nil
	}
}

func buildSyntaxRulesTransformer(name e.Identifier, syntaxRules e.List, defEnv *environment) (SyntaxTransformer, error) {
	// NewMacroGroup expects the full (define-syntax name (syntax-rules ...)) form
	defineSyntaxForm := e.Cons(e.Identifier("define-syntax"),
		e.Cons(name, e.Cons(syntaxRules, e.NIL)))

	mg, err := macromancy.NewMacroGroup(defineSyntaxForm)
	if err != nil {
		return SyntaxTransformer{}, err
	}

	macros := mg.Macros()

	definitionBindings := collectBoundIdentifiers(defEnv)

	return SyntaxTransformer{
		Transform: func(code e.List, mark macromancy.Mark) (e.Expr, error) {
			for _, m := range macros {
				if ok, bound := m.Matches(code); ok {
					return macromancy.ExpandHygienicWithDefinitionBindings(m.Body, bound, mark, m.PatternVars, definitionBindings), nil
				}
			}
			return nil, fmt.Errorf("no matching pattern for %s", code.Repr())
		},
	}, nil
}

func collectBoundIdentifiers(env *environment) map[e.Identifier]bool {
	// Special form keywords are recognised by the evaluator via Equiv checks,
	// so marking them would prevent recognition after expansion.
	result := map[e.Identifier]bool{
		COND_SPECIAL_FORM:          true,
		ELSE_SPECIAL_FORM:          true,
		BEGIN_SPECIAL_FORM:         true,
		LAMBDA_SPECIAL_FORM:       true,
		DEFINE_SPECIAL_FORM:       true,
		ASSIGNMENT_SPECIAL_FORM:   true,
		DEFINE_SYNTAX_SPECIAL_FORM: true,
		SYNTAX_RULES_FORM:         true,
	}
	for i := range *env {
		for key := range *(*env)[i] {
			if key.MarksKey == "" { // only plain identifiers
				result[e.Identifier(key.Name)] = true
			}
		}
	}
	return result
}

type EvaluationError struct {
	msg       string
	ErrorList e.List
}

func NewEvaluationError(msg string, errorList e.List) EvaluationError {
	return EvaluationError{msg, errorList}
}

func (err EvaluationError) Error() string {
	if pair, ok := err.ErrorList.(*e.Pair); ok && pair.Loc != nil {
		msg := fmt.Sprintf("%s: %s", pair.Loc.String(), err.msg)
		if ctx := pair.Loc.SourceContext(); ctx != "" {
			msg += "\n\n" + ctx
		}
		return msg
	}
	return err.msg
}
