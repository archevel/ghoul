package evaluator

import (
	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
)

const COND_SPECIAL_FORM = e.Identifier("cond")
const ELSE_SPECIAL_FORM = e.Identifier("else")
const BEGIN_SPECIAL_FORM = e.Identifier("begin")
const LAMBDA_SPECIAL_FORM = e.Identifier("lambda")
const DEFINE_SPECIAL_FORM = e.Identifier("define")
const ASSIGNMENT_SPECIAL_FORM = e.Identifier("set!")

type continuation func(arg e.Expr, ev *Evaluator) (e.Expr, error)
type contStack []continuation

func Evaluate(exprs e.Expr, env *environment) (res e.Expr, err error) {
	evaluator := New(logging.StandardLogger, env)
	return evaluator.Evaluate(exprs)
}

func New(logger logging.Logger, env *environment) *Evaluator {
	return &Evaluator{logger, env, nil}
}

type Evaluator struct {
	log   logging.Logger
	env   *environment
	conts *contStack
}

func (ev *Evaluator) Evaluate(exprs e.Expr) (e.Expr, error) {
	if exprs == e.NIL {
		return exprs, nil
	}
	listExpr := wrappNonList(exprs)
	ev.conts = &contStack{sexprSeqEvalContinuationFor(listExpr, false)}

	return ev.stepThroughContinuations()
}

func (ev *Evaluator) stepThroughContinuations() (e.Expr, error) {
	var ret e.Expr = e.NIL
	var err error

	ev.log.Debug("Starting to step through continuations")
	for len(*ev.conts) > 0 {
		next := ev.popContinuation()
		ret, err = next(ret, ev)

		if err != nil {
			ev.log.Debug("Continuation returned an error!")
			return nil, err
		}

	}

	ev.log.Debug("Nothing left to evaluate. Returning %s", ret)
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
			ev.log.Debug("Pushing continuation for evaluating tail of expression sequence")
			ev.pushContinuation(sexprSeqEvalContinuationFor(t, maybeTailCall))
		} else if !ok {
			return nil, NewEvaluationError("Malformed expresion sequence", exprs)
		}
		head := exprs.Head()
		ev.log.Debug("Pushing continuation for evaluating head of expression sequence")
		ev.pushContinuation(sexprEvalContinuationFor(head, exprs, maybeTailCall && t == e.NIL))
		return e.NIL, nil
	}
}

func sexprEvalContinuationFor(expr e.Expr, parent e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		ev.log.Debug("Choosing evaluation continuation")
		ret, nextCont, err := chooseEvaluation(expr, parent, maybeTailCall)
		if nextCont != nil {
			ev.log.Debug("Pushing choice")
			ev.pushContinuation(nextCont)
		}

		return ret, err
	}
}

func chooseEvaluation(expr e.Expr, parent e.List, maybeTailCall bool) (ret e.Expr, nextCont continuation, err error) {
	h, t, isList := maybeSplitExpr(expr)
	if quote, ok := expr.(*e.Quote); ok {
		ret = quote.Quoted
	} else if ident, ok := expr.(e.Identifier); ok {
		nextCont = makeIdentificationLookupContinuationFor(ident, parent)
		ret = e.NIL
	} else if !isList && h == nil {
		ret = expr
	} else if !isList {
		err = NewEvaluationError("Malformed expression", parent)
	} else if DEFINE_SPECIAL_FORM.Equiv(h) {
		nextCont = defineContinuationFor(t, maybeTailCall)
		ret = e.NIL
	} else if LAMBDA_SPECIAL_FORM.Equiv(h) {
		nextCont = lambdaContinuationFor(t)
		ret = e.NIL
	} else if COND_SPECIAL_FORM.Equiv(h) {
		nextCont = conditionalContinuationFor(t, maybeTailCall)
		ret = e.NIL
	} else if ASSIGNMENT_SPECIAL_FORM.Equiv(h) {
		nextCont = assignmentContinuationFor(t, maybeTailCall)
		ret = e.NIL
	} else if BEGIN_SPECIAL_FORM.Equiv(h) {
		nextCont = sexprSeqEvalContinuationFor(t, maybeTailCall)
		ret = e.NIL
	} else {
		nextCont = functionCallContinuationFor(expr.(e.List), maybeTailCall)
		ret = e.NIL
	}

	return
}

func makeIdentificationLookupContinuationFor(ident e.Identifier, parent e.List) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		var env *environment = ev.env

		ev.log.Debug("Looking up identifier: %s", ident)
		resExpr, err := lookupIdentifier(ident, env)
		if err != nil {
			ev.log.Debug("Failed looking up identifier: %s", ident)
			err = NewEvaluationError(err.Error(), parent)
			return e.NIL, err
		}
		return resExpr, nil

	}
}
func assignmentContinuationFor(assignment e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		valueExpr, val_ok := assignment.Tail()
		if !val_ok {
			ev.log.Debug("Failed evaluate assignment: %s", assignment)
			return e.NIL, NewEvaluationError("Malformed assignment", assignment)
		}
		nilTail, nil_ok := valueExpr.Tail()
		if val_ok && nil_ok && valueExpr != e.NIL && nilTail == e.NIL {
			ev.log.Debug("Pushing anonymous assigment func: %s", assignment)
			ev.pushContinuation(func(value e.Expr, ev *Evaluator) (e.Expr, error) {
				var env *environment = ev.env

				ret, err := assign(assignment.Head(), value, env)
				return ret, err
			})
			ev.pushContinuation(sexprEvalContinuationFor(valueExpr.Head(), valueExpr, maybeTailCall))
			return e.NIL, nil

		} else {
			ev.log.Debug("Tail part of assignment expession was malformed: %s", assignment)
			return e.NIL, NewEvaluationError("Malformed assignment", assignment)
		}
	}

}

func conditionalContinuationFor(conds e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		if conds == e.NIL {
			ev.log.Debug("No condition successfully matched so returning NIL")
			return e.NIL, nil
		}

		alternative, ok := headList(conds)
		if !ok {
			ev.log.Debug("Malformed alternative of cond list. Head should be list, but was %s", conds.Head())
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+conds.Head().Repr(), conds)
		}

		if alternative == e.NIL {
			ev.log.Debug("Malformed alternative of cond list. Alternative was NIL")
			return nil, NewEvaluationError("Bad syntax: Missing condition", conds)
		}

		consequent, ok := alternative.Tail()
		if !ok {
			ev.log.Debug("Malformed alternative of cond list. Alternative likely in invalid pair form: %s", alternative)
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+alternative.Repr(), alternative)
		}

		if consequent == e.NIL {
			ev.log.Debug("Malformed alternative of cond list. Alternative tail was NIL in: %s", alternative)
			return nil, NewEvaluationError("Bad syntax: Missing consequent", alternative)
		}

		predExpr := alternative.Head()
		if predExpr.Equiv(ELSE_SPECIAL_FORM) {
			predExpr = e.Boolean(true)
		}

		nextPredOrConsequent := func(truthy e.Expr, ev *Evaluator) (e.Expr, error) {
			if isTruthy(truthy) {
				ev.log.Debug("Found truthy alternative pushing evaluation of consequent")
				ev.pushContinuation(sexprEvalContinuationFor(consequent.Head(), conds, maybeTailCall))
				return e.NIL, nil
			}

			tailConds, ok := conds.Tail()
			if !ok {
				ev.log.Debug("Malformed cond list. Tail was not a list in: %s", conds)
				return nil, NewEvaluationError("Bad syntax: Malformed cond, expected list not pair", conds)
			}

			ev.log.Debug("Trying next alternative in cond list")
			ev.pushContinuation(conditionalContinuationFor(tailConds, maybeTailCall))
			return e.NIL, nil
		}

		ev.log.Debug("Pushing evaluation of nextPredOrConsequent followed by evaluation of current alternative")
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
				ev.log.Debug("Pushing evaluation of lambda body and preparation of the lambdas scope")
				ev.pushContinuation(sexprSeqEvalContinuationFor(body, true))
				ev.pushContinuation(prepareScope(lambda.Head(), args, env))

				return e.NIL, nil
			}

			ev.log.Debug("Yielding Function expression value for lambda")
			return Function{&fun}, nil
		} else {
			ev.log.Debug("Lambda expression had malformed body: %s", lambda)
			return e.NIL, NewEvaluationError("Malformed lambda expression", lambda)
		}
	}
}

func prepareScope(paramExpr e.Expr, args e.List, definitionEnv *environment) continuation {

	return func(ignore e.Expr, ev *Evaluator) (e.Expr, error) {
		ev.log.Debug("Preparing new scope for function evaluation")
		newEnv := newEnvWithEmptyScope(definitionEnv)

		paramList, ok := paramExpr.(e.List)
		var variadicParam e.Expr = paramExpr

		ev.log.Debug("Binding function arguments %s to parameter list %s", args, paramList)
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
			ev.log.Debug("Binding remaining args %s to variadic parameter %s", args, variadicId)
			bindIdentifier(variadicId, args, newEnv)
		} else if args != e.NIL {
			ev.log.Debug("More arguments given than the function supports!")
			return e.NIL, NewEvaluationError("Arity mismatch: too many arguments", args)
		} else if paramList != e.NIL {
			ev.log.Debug("Not all parameters could be given a value!")
			return e.NIL, NewEvaluationError("Arity mismatch: too few arguments", args)
		}

		ev.log.Debug("Reassigning evaluator environment pointer to new environment")
		ev.env = newEnv
		return e.NIL, nil
	}
}

func isCall(expr e.Expr) (e.List, bool) {
	if list, ok := expr.(e.List); ok {
		return list, true
	}
	return nil, false
}

func functionCallContinuationFor(callable e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		ev.log.Debug("Evaluating function call")
		var callEnv *environment = ev.env

		if callable == e.NIL {
			ev.log.Debug("NIL is not a function that can be called!")
			return e.NIL, NewEvaluationError("Missing procedure expression in: ()", callable)
		}
		if !maybeTailCall {
			ev.log.Debug("Pushing environment restoration to evaluate after function call since the call is not made in tail position")
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
				ev.log.Debug("Can not apply a non-function!")
				return e.NIL, NewEvaluationError("Not a procedure: "+arg.Repr(), callable)
			}
			proc := funExpr.Fun

			ev.log.Debug("Applying function with arguments collected")
			res, err := (*proc)(argList, ev)
			return res, err
		}
		ev.log.Debug("Pushing function application and function resolution")
		ev.pushContinuation(applyFunc)
		resolveFunc := sexprEvalContinuationFor(callable.Head(), callable, false)
		ev.pushContinuation(resolveFunc)

		funcArgs, ok := callable.Tail()
		ev.log.Debug("Pushing collection and evaluation of function arguments")
		for ok && funcArgs != e.NIL {
			anArg := funcArgs.Head()
			ev.pushContinuation(collectArgs)
			ev.pushContinuation(sexprEvalContinuationFor(anArg, callable, false))
			funcArgs, ok = funcArgs.Tail()
			if !ok {
				ev.log.Debug("Function call is malformed in: %s", callable)
				return e.NIL, NewEvaluationError("Bad syntax in procedure application", funcArgs)
			}
		}
		return e.NIL, nil
	}
}

func defineContinuationFor(def e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		valueExpr, val_ok := def.Tail()
		if !val_ok {
			ev.log.Debug("Define has inproper fromat: %s", def)
			return nil, NewEvaluationError("Bad syntax: invalid binding format", def)
		}

		if valueExpr == e.NIL {
			ev.log.Debug("Define was not given a value to bind to identifier in: %s", def)
			return nil, NewEvaluationError("Bad syntax: missing value in binding", def)
		}
		if t, ok := valueExpr.Tail(); ok && t != e.NIL {
			ev.log.Debug("Define was given more than one argument after binding identifier: %s", def)
			return nil, NewEvaluationError("Bad syntax: multiple values in binding", def)
		}

		ev.log.Debug("Pushing binding of variable and evaluation of definition value")
		ev.pushContinuation(bindVar(def.Head()))
		ev.pushContinuation(sexprEvalContinuationFor(valueExpr.Head(), valueExpr, maybeTailCall))
		return e.NIL, nil
	}
}

func bindVar(expr e.Expr) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		var env *environment = ev.env
		ev.log.Debug("Binding identifier: %s to: %s", expr, arg)
		res, err := bindIdentifier(expr, arg, env)
		return res, err
	}
}

type EvaluationError struct {
	msg       string
	ErrorList e.List
}

func NewEvaluationError(msg string, errorList e.List) EvaluationError {
	return EvaluationError{msg, errorList}
}

func (err EvaluationError) Error() string {
	return err.msg
}
