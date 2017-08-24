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

func matchWalk(macro e.Expr, code e.Expr, bound bindings, hasElispsis bool) (bool, bindings) {
	if id, ok := macro.(e.Identifier); ok {
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
			if macroList == e.NIL || (codeList == e.NIL && (!hasElispsis || macroLength > 1)) {
				return false, nil
			}

			macHead := macroList.Head()
			if id, ok := macHead.(e.Identifier); ok && id.Equiv(e.Identifier("...")) {
				if macroList.Tail() == e.NIL {
					return matchWalk(macHead, codeList, bound, true)
				}
				followingPatternCount := macroLength - 1
				bindToMacHead, rest := splitListAt(followingPatternCount, codeList)
				headMatch, bound := matchWalk(macHead, bindToMacHead, bound, true)
				if headMatch {
					return matchWalk(macroList.Tail(), rest, bound, true)
				}
			} else {

				headMatch, bound := matchWalk(macHead, codeList.Head(), bound, hasElispsis)
				if headMatch {
					return matchWalk(macroList.Tail(), codeList.Tail(), bound, hasElispsis)
				}
			}
		} else {
			if macroList != e.NIL && macroList.Tail() == e.NIL {
				return matchWalk(macroList.Head(), code, bound, hasElispsis)
			}
			if macroList.Head().Equiv(e.Identifier("...")) {
				bound[macroList.Head().(e.Identifier)] = e.NIL
				if macroList != e.NIL {
					return matchWalk(macroList.Tail(), code, bound, hasElispsis)
				}
			}

		}
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
	for i := 0; i < count-1; i++ {
		if sp, ok := splitPoint.Tail().(e.List); ok {
			splitPoint = sp
		} else {
			break
		}

	}

	rest, ok = splitPoint.Tail().(e.List)
	if (rest == e.NIL || !ok) && count > 1 {
		return e.NIL, codeList
	}

	if splitPoint != e.NIL && (ok || count == 1) {
		rest = splitPoint.Tail()
		splitPoint.(*e.Pair).T = e.NIL
	} else {
		rest = e.NIL
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
