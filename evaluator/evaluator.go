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
	evaluator := Evaluator{env, nil}
	return evaluator.Evaluate(exprs)
}

func NewEvaluator(logger logging.Logger, env *environment) *Evaluator {
	return &Evaluator{env, nil}
}

type Evaluator struct {
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

	for len(*ev.conts) > 0 {
		next := ev.popContinuation()
		ret, err = next(ret, ev)

		if err != nil {
			return nil, err
		}

	}

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
			ev.pushContinuation(sexprSeqEvalContinuationFor(t, maybeTailCall))
		} else if !ok {
			return nil, NewEvaluationError("Malformed expresion sequence", exprs)
		}

		ev.pushContinuation(sexprEvalContinuationFor(exprs.Head(), exprs, maybeTailCall && t == e.NIL))
		return e.NIL, nil
	}
}

func sexprEvalContinuationFor(expr e.Expr, parent e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		ret, nextCont, err := chooseEvaluation(expr, parent, maybeTailCall)
		if nextCont != nil {
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

		resExpr, err := lookupIdentifier(ident, env)
		if err != nil {
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
			return e.NIL, NewEvaluationError("Malformed assignment", assignment)
		}
		nilTail, nil_ok := valueExpr.Tail()
		if val_ok && nil_ok && valueExpr != e.NIL && nilTail == e.NIL {
			ev.pushContinuation(func(value e.Expr, ev *Evaluator) (e.Expr, error) {
				var env *environment = ev.env

				ret, err := assign(assignment.Head(), value, env)
				return ret, err
			})
			ev.pushContinuation(sexprEvalContinuationFor(valueExpr.Head(), valueExpr, maybeTailCall))
			return e.NIL, nil

		} else {
			return e.NIL, NewEvaluationError("Malformed assignment", assignment)
		}
	}

}

func conditionalContinuationFor(conds e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		if conds == e.NIL {
			return e.NIL, nil
		}

		alternative, ok := headList(conds)
		if !ok {
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+conds.Head().Repr(), conds)
		}

		if alternative == e.NIL {
			return nil, NewEvaluationError("Bad syntax: Missing condition", conds)
		}

		consequent, ok := alternative.Tail()
		if !ok {
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+alternative.Repr(), alternative)
		}

		if consequent == e.NIL {
			return nil, NewEvaluationError("Bad syntax: Missing consequent", alternative)
		}

		predExpr := alternative.Head()
		if predExpr.Equiv(ELSE_SPECIAL_FORM) {
			predExpr = e.Boolean(true)
		}

		nextPredOrConsequent := func(truthy e.Expr, ev *Evaluator) (e.Expr, error) {
			if isTruthy(truthy) {
				ev.pushContinuation(sexprEvalContinuationFor(consequent.Head(), conds, maybeTailCall))
				return e.NIL, nil
			}

			tailConds, ok := conds.Tail()
			if !ok {
				return nil, NewEvaluationError("Bad syntax: Malformed cond, expected list not pair", conds)
			}

			ev.pushContinuation(conditionalContinuationFor(tailConds, maybeTailCall))
			return e.NIL, nil
		}

		ev.pushContinuation(nextPredOrConsequent)
		ev.pushContinuation(sexprEvalContinuationFor(predExpr, alternative, false))

		return e.NIL, nil
	}

}

func lambdaContinuationFor(lambda e.List) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		var env *environment = ev.env
		if body, ok := lambda.Tail(); ok {
			fun := func(args e.List) (e.Expr, error) {
				// evaluate body
				ev.pushContinuation(sexprSeqEvalContinuationFor(body, true))
				// bind params in new scope
				ev.pushContinuation(prepareScope(lambda.Head(), args, env))

				return e.NIL, nil
			}
			return e.Function{&fun}, nil
		} else {
			return e.NIL, NewEvaluationError("Malformed lambda expression", lambda)
		}
	}
}

func prepareScope(paramExpr e.Expr, args e.List, definitionEnv *environment) continuation {

	return func(ignore e.Expr, ev *Evaluator) (e.Expr, error) {
		newEnv := newEnvWithEmptyScope(definitionEnv)

		paramList, ok := paramExpr.(e.List)
		var variadicParam e.Expr = paramExpr

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
			bindIdentifier(variadicId, args, newEnv)
		} else if args != e.NIL {
			return e.NIL, NewEvaluationError("Arity mismatch: too many arguments", args)
		} else if paramList != e.NIL {
			return e.NIL, NewEvaluationError("Arity mismatch: too few arguments", args)
		}

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
		var callEnv *environment = ev.env

		if callable == e.NIL {
			return e.NIL, NewEvaluationError("Missing procedure expression in: ()", callable)
		}
		if !maybeTailCall {
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
			funExpr, ok := arg.(e.Function)
			if !ok {
				return e.NIL, NewEvaluationError("Not a procedure: "+arg.Repr(), callable)
			}
			proc := funExpr.Fun
			res, err := (*proc)(argList)
			return res, err
		}
		ev.pushContinuation(applyFunc)

		resolveFunc := sexprEvalContinuationFor(callable.Head(), callable, false)
		ev.pushContinuation(resolveFunc)

		funcArgs, ok := callable.Tail()
		for ok && funcArgs != e.NIL {
			anArg := funcArgs.Head()
			ev.pushContinuation(collectArgs)
			ev.pushContinuation(sexprEvalContinuationFor(anArg, callable, false))
			funcArgs, ok = funcArgs.Tail()
			if !ok {
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
			return nil, NewEvaluationError("Bad syntax: invalid binding format", def)
		}

		if valueExpr == e.NIL {
			return nil, NewEvaluationError("Bad syntax: missing value in binding", def)
		}
		if t, ok := valueExpr.Tail(); ok && t != e.NIL {

			return nil, NewEvaluationError("Bad syntax: multiple values in binding", def)
		}

		ev.pushContinuation(bindVar(def.Head()))
		ev.pushContinuation(sexprEvalContinuationFor(valueExpr.Head(), valueExpr, maybeTailCall))
		return e.NIL, nil
	}
}

func bindVar(expr e.Expr) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		var env *environment = ev.env

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
