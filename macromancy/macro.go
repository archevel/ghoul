package macromancy

import (
	e "github.com/archevel/ghoul/expressions"
)

type Macro struct {
	Pattern e.Expr
}

func (m Macro) Matches(expr e.Expr) bool {
	toMatch, macroArgCount := identifierAndArgCount(m.Pattern)
	candidate, codeArgCount := identifierAndArgCount(expr)
	return candidate.Equiv(toMatch) && macroArgCount == codeArgCount
}

func identifierAndArgCount(expr e.Expr) (e.Expr, int) {
	identifier := expr
	argCount := -1

	if list, ok := expr.(e.List); ok {
		identifier = list.Head()
		argCount = 0
		for ; ok && list.Tail() != e.NIL; list, ok = list.Tail().(e.List) {
			argCount++
		}
	}

	return identifier, argCount
}
