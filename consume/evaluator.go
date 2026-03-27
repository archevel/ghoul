// Package evaluator implements a continuation-passing style (CPS) expression
// evaluator with proper tail call optimization. Evaluation proceeds by pushing
// continuations onto a stack and stepping through them in a trampoline loop
// (stepThroughContinuationsWithContext). Each continuation receives the result
// of the previous one and the evaluator state, returning the next result.
//
// Special forms (cond, begin, lambda, define, set!, quote, require)
// are dispatched in chooseEvaluation. Function calls go through
// functionCallContinuationFor.
package consume

import (
	"context"
	"errors"
	"fmt"

	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
)

const COND_SPECIAL_FORM = e.Identifier("cond")
const ELSE_SPECIAL_FORM = e.Identifier("else")
const BEGIN_SPECIAL_FORM = e.Identifier("begin")
const LAMBDA_SPECIAL_FORM = e.Identifier("lambda")
const DEFINE_SPECIAL_FORM = e.Identifier("define")
const ASSIGNMENT_SPECIAL_FORM = e.Identifier("set!")
const REQUIRE_SPECIAL_FORM = e.Identifier("require")
const QUOTE_SPECIAL_FORM = e.Identifier("quote")

type continuation func(arg e.Expr, ev *Evaluator) (e.Expr, error)
type contStack []continuation

func Evaluate(exprs e.Expr, env *environment) (res e.Expr, err error) {
	return EvaluateWithContext(context.Background(), exprs, env)
}

func EvaluateWithContext(ctx context.Context, exprs e.Expr, env *environment) (res e.Expr, err error) {
	evaluator := New(engraving.StandardLogger, env)
	return evaluator.EvaluateWithContext(ctx, exprs)
}

func New(logger engraving.Logger, env *environment) *Evaluator {
	var counter uint64
	return NewWithMarkCounter(logger, env, &counter)
}

// NewWithMarkCounter creates an evaluator sharing an external mark counter.
// This allows the expansion phase and evaluation phase to share hygiene
// marks so that identifiers marked during expansion are recognized during
// evaluation.
func NewWithMarkCounter(logger engraving.Logger, env *environment, markCounter *uint64) *Evaluator {
	return &Evaluator{log: logger, env: env, requiredModules: map[string]bool{}, markCounter: markCounter}
}

type Evaluator struct {
	log              engraving.Logger
	env              *environment
	conts            *contStack
	requiredModules  map[string]bool
	moduleState      *ModuleState
	markCounter      *uint64
}

func (ev *Evaluator) Evaluate(exprs e.Expr) (e.Expr, error) {
	return ev.EvaluateWithContext(context.Background(), exprs)
}

// EvalSubExpression evaluates a single expression in the current environment,
// using a fresh continuation stack. Used by stdlib functions like map/filter
// that need to call Ghoul functions from Go code.
func (ev *Evaluator) EvalSubExpression(expr e.Expr) (e.Expr, error) {
	subEval := &Evaluator{
		log:             ev.log,
		env:             ev.env,
		requiredModules: ev.requiredModules,
		moduleState:     ev.moduleState,
		markCounter:     ev.markCounter,
	}
	return subEval.Evaluate(e.Cons(expr, e.NIL))
}

func (ev *Evaluator) SetModuleState(ms *ModuleState) {
	ev.moduleState = ms
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

// sexprSeqEvalContinuationFor evaluates a sequence of expressions left to right.
// maybeTailCall indicates this sequence is in tail position of its enclosing
// body — when true, the last expression can reuse the caller's stack frame
// (tail call optimization: no environment restoration is pushed after the call).
func sexprSeqEvalContinuationFor(exprs e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		t, ok := exprs.Tail()
		if ok && t != e.NIL {
			ev.log.Trace("Pushing continuation for evaluating tail of expression sequence")
			ev.pushContinuation(sexprSeqEvalContinuationFor(t, maybeTailCall))
		} else if !ok {
			return nil, NewEvaluationError("Malformed expression sequence", exprs)
		}
		head := exprs.First()
		isTail := maybeTailCall && t == e.NIL
		ev.log.Trace("Pushing continuation for evaluating head of expression sequence")
		ev.pushContinuation(sexprEvalContinuationFor(head, exprs, isTail))
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
// matching, ignoring hygiene marks on ScopedIdentifiers.
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
		case REQUIRE_SPECIAL_FORM:
			nextCont = requireContinuationFor(t)
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
		case QUOTE_SPECIAL_FORM:
			// (quote expr) returns expr unevaluated, like the parser's ' syntax
			if t != e.NIL {
				ret = t.First()
			}
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
			err = WrapError(err.Error(), parent, err)
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
			ev.log.Trace("Pushing anonymous assignment func: %s", assignment)
			ev.pushContinuation(func(value e.Expr, ev *Evaluator) (e.Expr, error) {
				var env *environment = ev.env

				ret, err := assign(assignment.First(), value, env)
				return ret, err
			})
			ev.pushContinuation(sexprEvalContinuationFor(valueExpr.First(), valueExpr, maybeTailCall))
			return e.NIL, nil

		} else {
			ev.log.Trace("Tail part of assignment expression was malformed: %s", assignment)
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
			ev.log.Trace("Malformed alternative of cond list. Head should be list, but was %s", conds.First())
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+conds.First().Repr(), conds)
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

		predExpr := alternative.First()
		if specialFormName(predExpr) == ELSE_SPECIAL_FORM {
			predExpr = e.Boolean(true)
		}

		nextPredOrConsequent := func(truthy e.Expr, ev *Evaluator) (e.Expr, error) {
			if isTruthy(truthy) {
				ev.log.Trace("Found truthy alternative pushing evaluation of consequent")
				ev.pushContinuation(sexprSeqEvalContinuationFor(consequent, maybeTailCall))
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
				ev.pushContinuation(prepareScope(lambda.First(), args, env))

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
			arg := args.First()
			args, _ = args.Tail()
			param := paramList.First()
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

// functionCallContinuationFor sets up the continuation stack for a function call.
// Continuations are pushed in reverse execution order (stack-based):
//
//  1. Environment restore (skipped for tail calls)
//  2. Apply function to collected arguments
//  3. Resolve function identifier
//  4. For each argument (right to left): collect result, then evaluate
//
// After the expansion phase, all macro calls have been resolved. The evaluator
// only handles core forms and regular function calls.
func functionCallContinuationFor(callable e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		ev.log.Trace("Evaluating function call")

		if callable == e.NIL {
			ev.log.Trace("NIL is not a function that can be called!")
			return e.NIL, NewEvaluationError("Missing procedure expression in: ()", callable)
		}

		var callEnv *environment = ev.env

		if !maybeTailCall {
			ev.log.Trace("Pushing environment restoration to evaluate after function call since the call is not made in tail position")
			ev.pushContinuation(func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
				ev.env = callEnv
				return arg, nil
			})
		}

		// collectArgs accumulates each evaluated argument into argList.
		// Arguments are evaluated right-to-left (stack order) and consed
		// onto the front, so argList ends up in the correct left-to-right order.
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
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return res, err
				}
				return res, WrapError(err.Error(), callable, err)
			}
			return res, nil
		}
		ev.log.Trace("Pushing function application and function resolution")
		ev.pushContinuation(applyFunc)
		resolveFunc := sexprEvalContinuationFor(callable.First(), callable, false)
		ev.pushContinuation(resolveFunc)

		funcArgs, ok := callable.Tail()
		ev.log.Trace("Pushing collection and evaluation of function arguments")
		for ok && funcArgs != e.NIL {
			anArg := funcArgs.First()
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
			ev.log.Trace("Define has improper format: %s", def)
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
		ev.pushContinuation(bindVar(def.First()))
		ev.pushContinuation(sexprEvalContinuationFor(valueExpr.First(), valueExpr, maybeTailCall))
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




type EvaluationError struct {
	msg       string
	ErrorList e.List
	cause     error
}

func NewEvaluationError(msg string, errorList e.List) EvaluationError {
	return EvaluationError{msg: msg, ErrorList: errorList}
}

func WrapError(msg string, errorList e.List, cause error) EvaluationError {
	return EvaluationError{msg: msg, ErrorList: errorList, cause: cause}
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

func (err EvaluationError) Unwrap() error {
	return err.cause
}
