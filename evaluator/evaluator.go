package evaluator

import (
	e "github.com/archevel/ghoul/expressions"
)

// TODO: Make error messages contain line and column of failed expression. Derived expressions should as far as possible point to their original version.
// TODO: Use fn instead of lambda?
// TODO: Use do instead of begin?
// TODO: Use def instead of define?
// TODO: Implement macros. Should they be hygenic?
// TODO: Clean up tests into separate files with distinct areas
// TODO: Implement an error printer

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
		t, ok := tail(exprs)
		if ok && t != e.NIL {
			ev.pushContinuation(sexprSeqEvalContinuationFor(t, maybeTailCall))
		} else if !ok {
			return nil, NewEvaluationError("Malformed expresion sequence", exprs)
		}

		ev.pushContinuation(sexprEvalContinuationFor(head(exprs), exprs, maybeTailCall && t == e.NIL))
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
		valueExpr, val_ok := tail(assignment)
		if !val_ok {
			return e.NIL, NewEvaluationError("Malformed assignment", assignment)
		}
		nilTail, nil_ok := tail(valueExpr)
		if val_ok && nil_ok && valueExpr != e.NIL && nilTail == e.NIL {
			ev.pushContinuation(func(value e.Expr, ev *Evaluator) (e.Expr, error) {
				var env *environment = ev.env

				ret, err := assign(head(assignment), value, env)
				return ret, err
			})
			ev.pushContinuation(sexprEvalContinuationFor(head(valueExpr), valueExpr, maybeTailCall))
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
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+head(conds).Repr(), conds)
		}

		if alternative == e.NIL {
			return nil, NewEvaluationError("Bad syntax: Missing condition", conds)
		}

		consequent, ok := tail(alternative)
		if !ok {
			return nil, NewEvaluationError("Bad syntax: Malformed cond clause: "+alternative.Repr(), alternative)
		}

		if consequent == e.NIL {
			return nil, NewEvaluationError("Bad syntax: Missing consequent", alternative)
		}

		predExpr := head(alternative)
		if predExpr.Equiv(ELSE_SPECIAL_FORM) {
			predExpr = e.Boolean(true)
		}

		nextPredOrConsequent := func(truthy e.Expr, ev *Evaluator) (e.Expr, error) {
			if isTruthy(truthy) {
				ev.pushContinuation(sexprEvalContinuationFor(head(consequent), conds, maybeTailCall))
				return e.NIL, nil
			}

			tailConds, ok := tail(conds)
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
		if body, ok := tail(lambda); ok {
			fun := func(args e.List) (e.Expr, error) {
				// evaluate body
				ev.pushContinuation(sexprSeqEvalContinuationFor(body, true))
				// bind params in new scope
				ev.pushContinuation(prepareScope(head(lambda), args, env))

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
			arg := head(args)
			args, _ = tail(args)
			param := head(paramList)
			pl, ok := tail(paramList)
			if !ok {
				variadicParam = paramList.Tail()
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

		resolveFunc := sexprEvalContinuationFor(head(callable), callable, false)
		ev.pushContinuation(resolveFunc)

		funcArgs, ok := tail(callable)
		for ok && funcArgs != e.NIL {
			anArg := head(funcArgs)
			ev.pushContinuation(collectArgs)
			ev.pushContinuation(sexprEvalContinuationFor(anArg, callable, false))
			funcArgs, ok = tail(funcArgs)
			if !ok {
				return e.NIL, NewEvaluationError("Bad syntax in procedure application", funcArgs)
			}
		}
		return e.NIL, nil
	}
}

func defineContinuationFor(def e.List, maybeTailCall bool) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		valueExpr, val_ok := tail(def)
		if !val_ok {
			return nil, NewEvaluationError("Bad syntax: invalid binding format", def)
		}

		if valueExpr == e.NIL {
			return nil, NewEvaluationError("Bad syntax: missing value in binding", def)
		}
		if t, ok := tail(valueExpr); ok && t != e.NIL {

			return nil, NewEvaluationError("Bad syntax: multiple values in binding", def)
		}

		ev.pushContinuation(bindVar(head(def)))
		ev.pushContinuation(sexprEvalContinuationFor(head(valueExpr), valueExpr, maybeTailCall))
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
