package macromancy

import (
	e "github.com/archevel/ghoul/expressions"
)

type Macro struct {
	Pattern e.Expr
}

func (m Macro) Matches(expr e.Expr) (bool, map[e.Identifier]e.Expr) {
	toMatch, macroArgs := identifierAndArgs(m.Pattern)
	candidate, codeArgs := identifierAndArgs(expr)
	if candidate.Equiv(toMatch) && argsSeemOk(macroArgs, codeArgs) {
		return true, zipArgs(macroArgs, codeArgs)
	}
	return false, nil
}

func identifierAndArgs(expr e.Expr) (e.Expr, []e.Expr) {
	identifier := expr
	var argExprs []e.Expr = nil
	if list, ok := expr.(e.List); ok {
		identifier = list.Head()
		argExprs = []e.Expr{}
		for ; ok && list.Tail() != e.NIL; list, ok = list.Tail().(e.List) {
			argExprs = append(argExprs, list.Head())
		}
	}

	return identifier, argExprs
}

func zipArgs(macroArgs []e.Expr, codeArgs []e.Expr) map[e.Identifier]e.Expr {
	bindings := map[e.Identifier]e.Expr{}
	for i, mArg := range macroArgs {
		identifier := mArg.(e.Identifier)
		bindings[identifier] = codeArgs[i]
	}
	return bindings
}

func argsSeemOk(macroArgs []e.Expr, codeArgs []e.Expr) bool {
	return (macroArgs == nil && codeArgs == nil) || (macroArgs != nil && codeArgs != nil && len(macroArgs) == len(codeArgs))
}
