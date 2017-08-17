package macromancy

import (
	e "github.com/archevel/ghoul/expressions"
)

type bindings map[e.Identifier]e.Expr

type Macro struct {
	Pattern e.Expr
	Body    e.Expr
}

func (m Macro) Matches(expr e.Expr) (bool, bindings) {
	toMatch, macroArgs := identifierAndArgs(m.Pattern)

	candidate, codeArgs := identifierAndArgs(expr)
	if candidate.Equiv(toMatch) && argsSeemOk(macroArgs, codeArgs) {
		return true, zipArgs(macroArgs, codeArgs)
	}
	return false, nil
}

func (m Macro) Expand(bound bindings) (e.Expr, error) {
	toWalk := m.Body
	res := walkAndReplace(toWalk, bound)
	return res, nil
}

func walkAndReplace(toWalk e.Expr, bound bindings) e.Expr {

	if id, ok := toWalk.(e.Identifier); ok {
		replacement, present := bound[id]

		if present {
			return replacement
		}

	}

	if list, ok := toWalk.(e.List); ok && list != e.NIL {
		h := list.Head()
		return &e.Pair{walkAndReplace(h, bound), walkAndReplace(list.Tail(), bound)}
	}

	return toWalk
}

func identifierAndArgs(expr e.Expr) (e.Expr, []e.Expr) {
	identifier := expr
	var argExprs []e.Expr = nil
	if list, ok := expr.(e.List); ok {
		identifier = list.Head()
		argExprs = []e.Expr{}
		list = list.Tail().(e.List)

		for ; ok && list != e.NIL; list, ok = list.Tail().(e.List) {
			argExprs = append(argExprs, list.Head())
		}
	}

	return identifier, argExprs
}

func zipArgs(macroArgs []e.Expr, codeArgs []e.Expr) bindings {
	bindings := bindings{}
	for i, mArg := range macroArgs {
		identifier := mArg.(e.Identifier)
		bindings[identifier] = codeArgs[i]
	}
	return bindings
}

func argsSeemOk(macroArgs []e.Expr, codeArgs []e.Expr) bool {
	return (macroArgs == nil && codeArgs == nil) || (macroArgs != nil && codeArgs != nil && len(macroArgs) == len(codeArgs))
}
