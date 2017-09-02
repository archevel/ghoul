package evaluator

import (
	e "github.com/archevel/ghoul/expressions"
)

func head(expr e.List) e.Expr {
	return expr.First()
}

func headList(expr e.List) (list e.List, ok bool) {
	list, ok = expr.First().(e.List)
	return
}

func tail(expr e.List) (list e.List, ok bool) {
	list, ok = expr.Second().(e.List)
	return
}

func list(expr e.Expr, exprs ...e.Expr) e.List {
	var tail e.List = e.NIL
	for i := len(exprs) - 1; i >= 0; i-- {
		tail = e.Cons(exprs[i], tail)
	}
	return e.Cons(expr, tail)
}

func wrappNonList(expr e.Expr) e.List {
	if list, ok := expr.(e.List); ok {
		return list
	}

	return list(expr)
}

func cons(expr e.Expr, list e.List) e.List {
	return e.Cons(expr, list)
}

func isTruthy(truth e.Expr) bool {
	b, isBool := truth.(e.Boolean)
	return truth != e.NIL && (!isBool || bool(b))
}

func maybeSplitExpr(expr e.Expr) (e.Expr, e.List, bool) {
	if list, ok := expr.(e.List); ok {
		t, isList := tail(list)
		return head(list), t, isList
	}
	return nil, nil, false
}
