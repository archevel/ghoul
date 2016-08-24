package evaluator

import (
	"errors"
	e "github.com/archevel/ghoul/expressions"
	//	p "github.com/archevel/ghoul/parser"
	//	"fmt"
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
	res = expr
	listExpr := wrappNonList(expr)
	for listExpr != e.NIL {

		res, err = evaluateSexpr(head(listExpr), env)
		switch err.(type) {
		case EvaluationError:
			return nil, err
		case error:
			evErr := NewEvaluationError(err.Error(), listExpr.(*e.Pair))
			return nil, evErr
		}

		listExpr, _ = tail(listExpr)
	}

	return res, nil

}

func evaluateSexpr(expr e.Expr, env *environment) (e.Expr, error) {
	res := expr
	if def, ok := isOfKind(res, DEFINE_SPECIAL_FORM); ok {
		res, err := define(def, env)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	if assignment, ok := isOfKind(res, ASSIGNMENT_SPECIAL_FORM); ok {
		value, val_ok := tail(assignment)
		nilTail, nil_ok := tail(value)
		if val_ok && nil_ok && value != e.NIL && nilTail == e.NIL {
			res, err := makeAssignment(assignment, env)
			if err != nil {
				return nil, err
			}
			return res, nil
		}
	}

	if lambda, ok := isOfKind(res, LAMBDA_SPECIAL_FORM); ok {
		res := createLambda(lambda, env)
		return res, nil
	}

	if cond, ok := isCond(res); ok {
		res, err := evaluateCond(cond, env)
		return res, err
	}

	if begin, ok := isOfKind(res, BEGIN_SPECIAL_FORM); ok {
		res, err := evaluateBegin(begin, env)
		return res, err
	}

	if quote, ok := res.(*e.Quote); ok {
		return quote.Quoted, nil
	}

	if ident, ok := res.(e.Identifier); ok {
		res, err := lookupIdentifier(ident, env)
		return res, err
	}

	if callable, ok := isCall(res); ok {
		return call(callable, env)
	}

	return res, nil
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

func evaluateCond(conds e.List, env *environment) (e.Expr, error) {
	res := e.NIL
	for conds != e.NIL {
		alternative, ok := headList(conds)
		if !ok {
			return nil, errors.New("Bad syntax: Malformed cond clause: " + head(conds).Repr())
		}

		if alternative == e.NIL {
			return nil, errors.New("Bad syntax: Missing condition")
		}

		consequent, ok := tail(alternative)
		if !ok {
			return nil, errors.New("Bad syntax: Malformed cond clause: " + alternative.Repr())
		}

		if consequent == e.NIL {
			return nil, errors.New("Bad syntax: Missing consequent")
		}

		predExpr := head(alternative)
		if predExpr.Equiv(ELSE_SPECIAL_FORM) {
			predExpr = e.Boolean(true)
		}

		pred, err := evaluateSexpr(predExpr, env)
		if err != nil {
			return nil, err
		}

		if isTruthy(pred) {

			res, err := evaluateSexpr(head(consequent), env)
			return res, err
		}
		conds, ok = tail(conds)
		if !ok {
			return nil, errors.New("Bad syntax: Malformed cond, expected list not pair")
		}
	}

	return res, nil
}

func evaluateBegin(begin e.List, env *environment) (e.Expr, error) {
	return Evaluate(begin, env)
}

func createLambda(lambda e.List, env *environment) e.Function {
	fun := func(args e.List) (e.Expr, error) {
		//		fmt.Println("args:", args.Repr())
		new_env := newEnvWithEmptyFrame(env)
		paramList, ok := headList(lambda)
		//		fmt.Println("paramList:", paramList)
		var variadicParam e.Expr = head(lambda)

		for ok && paramList != e.NIL && args != e.NIL {
			arg := head(args)
			args, _ = tail(args)
			//			fmt.Println("now -- paramList:", paramList)
			param := head(paramList)
			//			fmt.Println("after head paramList:", paramList.Repr())
			pl, ok := tail(paramList)
			//			fmt.Println("after tail pl:", pl)
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
			return nil, errors.New("Arity mismatch: too many arguments")
		} else if paramList != e.NIL {
			return nil, errors.New("Arity mismatch: too few arguments")
		}

		return Evaluate(lambda.Tail(), new_env)
	}

	return e.Function{&fun}
}

func isCall(expr e.Expr) (e.List, bool) {
	if list, ok := expr.(e.List); ok {
		return list, true
	}
	return nil, false
}

func call(callable e.List, env *environment) (e.Expr, error) {
	if callable == e.NIL {
		return nil, errors.New("Missing procedure expression in: ()")
	}
	expr, err := evaluateSexpr(callable.Head(), env)
	if err != nil {
		return nil, err
	}
	funExpr, ok := expr.(e.Function)
	if !ok {
		return nil, errors.New("Not a procedure: " + expr.Repr())
	}
	proc := funExpr.Fun
	args, _ := tail(callable)

	evaledArgs, err := evaluateArguments(args, env)
	if err != nil {
		return nil, err
	}
	//	fmt.Println("evaledArg:" ,evaledArgs, "err", err)
	return (*proc)(evaledArgs)
}

func evaluateArguments(args e.List, env *environment) (e.List, error) {

	evaluatedInTail := e.Pair{nil, e.NIL}
	cur := &evaluatedInTail

	var ok bool = true
	for args != e.NIL {
		next := &e.Pair{}
		cur.T = next
		cur = next
		evArg, err := evaluateSexpr(head(args), env)
		if err != nil {
			return nil, err
		}
		cur.H = evArg
		cur.T = e.NIL
		args, ok = tail(args)
		if !ok {
			return nil, errors.New("Bad syntax in procedure application")

		}

	}

	evaluatedArgs, _ := tail(evaluatedInTail)
	return evaluatedArgs, nil
}

func define(def e.List, env *environment) (e.Expr, error) {

	valueExpr, val_ok := tail(def)
	if !val_ok {
		return nil, errors.New("Bad syntax: invalid binding format")
	}

	if valueExpr == e.NIL {
		return nil, errors.New("Bad syntax: missing value in binding")
	}

	if t, _ := tail(valueExpr); t != e.NIL {
		return nil, errors.New("Bad syntax: multiple values in binding")
	}

	value, err := evaluateSexpr(head(valueExpr), env)
	if err != nil {
		return nil, err
	}

	return bindIdentifier(head(def), value, env)

}

type EvaluationError struct {
	msg       string
	ErrorPair *e.Pair
}

func NewEvaluationError(msg string, errorPair *e.Pair) EvaluationError {
	return EvaluationError{msg, errorPair}
}

func (err EvaluationError) Error() string {
	return err.msg
}
