package evaluator

import (
	e "github.com/archevel/ghoul/expressions"
)

// TODO: Make error messages contain line and column of failed expression. Derived expressions should as far as possible point to their original version.
// TODO: Make Evaluate use proper tail call optimization if possible. This might be hard because it is not a golang feature... First try might be to see if we can log a message at tail possitions...
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

func Evaluate(expr e.Expr, env *environment) (res e.Expr, err error) {
	return evaluate(expr, env)
}

type continuation func(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error)
type contStack []continuation

func evaluate(exprs e.Expr, env *environment) (res e.Expr, err error) {
	if exprs == e.NIL {
		return exprs, nil
	}
	listExpr := wrappNonList(exprs)
	conts := &contStack{evaluateSexprSeq(listExpr)}
	var ret e.Expr = e.NIL

	for len(*conts) > 0 {
		cur := (*conts)[len(*conts)-1]
		*conts = (*conts)[:len(*conts)-1]
		ret, env, err = cur(ret, env, conts)
		if err != nil {
			return nil, err
		}

	}

	return ret, nil
}

func evaluateSexprSeq(exprs e.List) continuation {
	return func(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {

		if t, ok := tail(exprs); ok && t != e.NIL {
			*conts = append(*conts, evaluateSexprSeq(t))
		} else if !ok {
			return nil, nil, NewEvaluationError("Malformed expresion sequence", exprs)
		}
		*conts = append(*conts, evaluateSexpr(head(exprs)))
		*conts = *conts
		return e.NIL, env, nil
	}
}

func evaluateSexpr(expr e.Expr) continuation {
	return func(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {
		if def, ok := isOfKind(expr, DEFINE_SPECIAL_FORM); ok {
			err := define(def, conts)
			if err != nil {
				return nil, nil, err
			}
			return e.NIL, env, nil
		}

		if lambda, ok := isOfKind(expr, LAMBDA_SPECIAL_FORM); ok {
			*conts = append(*conts, createLambda(lambda))
			return e.NIL, env, nil
		}

		if conds, ok := isCond(expr); ok {
			*conts = append(*conts, conditional(conds))
			return e.NIL, env, nil
		}

		if assignment, ok := isOfKind(expr, ASSIGNMENT_SPECIAL_FORM); ok {
			value, val_ok := tail(assignment)
			nilTail, nil_ok := tail(value)
			if val_ok && nil_ok && value != e.NIL && nilTail == e.NIL {
				res, err := makeAssignment(assignment, env)
				if err != nil {
					return nil, env, err
				}
				return res, env, nil
			}
		}

		if quote, ok := expr.(*e.Quote); ok {
			return quote.Quoted, env, nil
		}

		if begin, ok := isOfKind(expr, BEGIN_SPECIAL_FORM); ok {
			*conts = append(*conts, evaluateSexprSeq(begin))
			return e.NIL, env, nil
		}

		if ident, ok := expr.(e.Identifier); ok {
			resExpr, err := lookupIdentifier(ident, env)
			return resExpr, env, err
		}

		if callable, ok := isCall(expr); ok {
			err := call(callable, conts)
			return e.NIL, env, err
		}

		return expr, env, nil
	}
}

func makeAssignment(assignment e.List, env *environment) (e.Expr, error) {
	id := head(assignment)
	t, _ := tail(assignment)

	return assign(id, head(t), env)
}

func isCond(expr e.Expr) (e.List, bool) {
	if list, ok := expr.(e.List); ok && list.Head().Equiv(COND_SPECIAL_FORM) {
		t, _ := tail(list)
		return t, true
	}
	return nil, false
}

func conditional(conds e.List) continuation {
	return func(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {
		if conds == e.NIL {
			return e.NIL, env, nil
		}

		alternative, ok := headList(conds)
		if !ok {
			return nil, env, NewEvaluationError("Bad syntax: Malformed cond clause: "+head(conds).Repr(), conds)
		}

		if alternative == e.NIL {
			return nil, env, NewEvaluationError("Bad syntax: Missing condition", conds)
		}

		consequent, ok := tail(alternative)
		if !ok {
			return nil, env, NewEvaluationError("Bad syntax: Malformed cond clause: "+alternative.Repr(), alternative)
		}

		if consequent == e.NIL {
			return nil, env, NewEvaluationError("Bad syntax: Missing consequent", alternative)
		}

		predExpr := head(alternative)
		if predExpr.Equiv(ELSE_SPECIAL_FORM) {
			predExpr = e.Boolean(true)
		}

		nextPredOrConsequent := func(truthy e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {
			if isTruthy(truthy) {
				*conts = append(*conts, evaluateSexpr(head(consequent)))
				return e.NIL, env, nil
			}

			tailConds, ok := tail(conds)
			if !ok {
				return nil, env, NewEvaluationError("Bad syntax: Malformed cond, expected list not pair", conds)
			}

			*conts = append(*conts, conditional(tailConds))
			return e.NIL, env, nil
		}

		*conts = append(*conts, nextPredOrConsequent)
		*conts = append(*conts, evaluateSexpr(predExpr))
		return e.NIL, env, nil
	}

}

func createLambda(lambda e.List) continuation {
	return func(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {
		fun := func(args e.List) (e.Expr, error) {
			// drop env scope back to before calling func
			*conts = append(*conts, dropScope)
			// evaluate body
			body, _ := tail(lambda)

			*conts = append(*conts, evaluateSexprSeq(body))
			// bind params in new scope
			*conts = append(*conts, createNewScopeWithBoundArgs(head(lambda), args, env))
			return e.NIL, nil
		}
		return e.Function{&fun}, env, nil
	}
}

func createNewScopeWithBoundArgs(paramExpr e.Expr, args e.List, env *environment) continuation {
	return func(ignore e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {
		new_env := newEnvWithEmptyScope(env)
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
			bindIdentifier(param, arg, new_env)
		}

		if variadicId, ok := variadicParam.(e.Identifier); ok {
			bindIdentifier(variadicId, args, new_env)
		} else if args != e.NIL {
			return e.NIL, new_env, NewEvaluationError("Arity mismatch: too many arguments", args)
		} else if paramList != e.NIL {
			return e.NIL, new_env, NewEvaluationError("Arity mismatch: too few arguments", args)
		}

		return e.NIL, new_env, nil
	}
}

func dropScope(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {
	callingEnv := (*env)[:len(*env)-1]
	return arg, &callingEnv, nil
}
func isCall(expr e.Expr) (e.List, bool) {
	if list, ok := expr.(e.List); ok {
		return list, true
	}
	return nil, false
}

func call(callable e.List, conts *contStack) error {
	if callable == e.NIL {
		return NewEvaluationError("Missing procedure expression in: ()", callable)
	}
	var argList e.List = e.NIL
	collectArgs := func(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {

		argList = cons(arg, argList)
		return argList, env, nil
	}

	applyFunc := func(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {

		funExpr, ok := arg.(e.Function)
		if !ok {
			return e.NIL, env, NewEvaluationError("Not a procedure: "+arg.Repr(), callable)
		}
		proc := funExpr.Fun
		res, err := (*proc)(argList)
		return res, env, err
	}
	*conts = append(*conts, applyFunc)

	resolveFunc := evaluateSexpr(head(callable))
	*conts = append(*conts, resolveFunc)

	funcArgs, ok := tail(callable)
	for ok && funcArgs != e.NIL {
		anArg := head(funcArgs)
		*conts = append(*conts, collectArgs)
		*conts = append(*conts, evaluateSexpr(anArg))
		funcArgs, ok = tail(funcArgs)
		if !ok {
			return NewEvaluationError("Bad syntax in procedure application", funcArgs)
		}
	}
	return nil
}

func define(def e.List, conts *contStack) error {
	valueExpr, val_ok := tail(def)
	if !val_ok {
		return NewEvaluationError("Bad syntax: invalid binding format", def)
	}

	if valueExpr == e.NIL {
		return NewEvaluationError("Bad syntax: missing value in binding", def)
	}

	if t, _ := tail(valueExpr); t != e.NIL {
		return NewEvaluationError("Bad syntax: multiple values in binding", def)
	}

	*conts = append(*conts, bindVar(head(def)))
	*conts = append(*conts, evaluateSexpr(head(valueExpr)))

	return nil
}

func bindVar(expr e.Expr) continuation {
	return func(arg e.Expr, env *environment, conts *contStack) (e.Expr, *environment, error) {
		res, err := bindIdentifier(expr, arg, env)
		return res, env, err
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
