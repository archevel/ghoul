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
	macId, macPat := idAndRest(m.Pattern)
	codeId, code := idAndRest(expr)
	if macId.Equiv(codeId) {
		if macPat == nil && code == nil {
			return true, bindings{}

		}
		return matchWalk(macPat, code, bindings{}, false)
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

func matchWalk(macro e.Expr, code e.Expr, bound bindings, hasElipsis bool) (bool, bindings) {
	if id, ok := macro.(e.Identifier); ok {
		b, present := bound[id]
		if present && !b.Equiv(code) {
			return false, bound
		}
		bound[id] = code
		return true, bound
	}
	if macroList, macroOk := macro.(e.List); macroOk {
		codeList, codeOk := code.(e.List)
		if codeOk {

			if macroList == e.NIL && codeList == e.NIL {
				return true, bound
			}
			macroLength, _ := listLength(macroList)
			if macroList == e.NIL || (codeList == e.NIL && (!hasElipsis || macroLength > 1)) {
				return false, nil
			}

			macHead := macroList.Head()
			if id, ok := macHead.(e.Identifier); ok && id.Equiv(e.Identifier("...")) {
				if macroList.Tail() == e.NIL {
					// bind ... to all of code (1 2 3)
					return matchWalk(macHead, codeList, bound, true)
				}

				// there is atleast one part of the pattern following ...
				followingPatternCount := macroLength - 1
				bindToMacHead, rest := splitListAt(followingPatternCount, codeList)
				_, bound := matchWalk(macHead, bindToMacHead, bound, true)

				return matchWalk(macroList.Tail(), rest, bound, true)
			} else {
				// head is not an elipsis
				headMatch, bound := matchWalk(macHead, codeList.Head(), bound, hasElipsis)
				if headMatch {
					return matchWalk(macroList.Tail(), codeList.Tail(), bound, hasElipsis)
				}
			}
		} else {
			// code is not a list so probably single value
			if macroList != e.NIL && macroList.Tail() == e.NIL {
				// there is a macro head and it is the last one
				return matchWalk(macroList.Head(), code, bound, hasElipsis)
			}
			if macroList.Head().Equiv(e.Identifier("...")) {
				// the macro head is ... and there are more macro patterns`
				bound[macroList.Head().(e.Identifier)] = e.NIL
				return matchWalk(macroList.Tail(), code, bound, hasElipsis)
			}

		}
	}
	if code != nil && macro != nil && code.Equiv(macro) {
		return true, bound
	}
	return false, nil
}

func idAndRest(expr e.Expr) (e.Identifier, e.Expr) {
	identifier := expr
	if list, ok := expr.(e.List); ok {
		identifier = list.Head().(e.Identifier)
		return identifier.(e.Identifier), list.Tail()

	}
	return identifier.(e.Identifier), nil
}

func splitListAt(count int, codeList e.List) (e.Expr, e.Expr) {

	firstN := codeList
	splitPoint := firstN
	var rest e.Expr
	ok := false
	codeLength, _ := listLength(splitPoint)
	target := codeLength - count - 1

	if target < 0 {
		return e.NIL, codeList
	}

	for i := 0; i < target; i++ {
		if sp, ok := splitPoint.Tail().(e.List); ok {
			splitPoint = sp
		} else {
			break
		}
	}
	rest, ok = splitPoint.Tail().(e.List)
	if splitPoint != e.NIL && (ok || count == 1) {
		rest = splitPoint.Tail()
		splitPoint.(*e.Pair).T = e.NIL
	}

	return firstN, rest
}

func listLength(list e.List) (int, bool) {
	count := 0
	ok := false
	for list != e.NIL {
		count = count + 1
		list, ok = list.Tail().(e.List)
		if !ok {
			return count + 1, false
		}
	}
	return count, true
}
