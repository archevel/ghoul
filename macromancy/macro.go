package macromancy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
)

type bindings map[e.Identifier]e.Expr

type Macro struct {
	Pattern     e.Expr
	Body        e.Expr
	PatternVars map[e.Identifier]bool
}

func (m Macro) Matches(expr e.Expr) (bool, bindings) {
	macId, macPat, err := idAndRest(m.Pattern)
	if err != nil {
		return false, nil
	}
	codeId, code, err := idAndRest(expr)
	if err != nil {
		return false, nil
	}
	if macId.Equiv(codeId) {
		if macPat == nil && code == nil {
			return true, bindings{}

		}
		return matchWalk(macPat, code, bindings{}, false)
	}
	return false, nil
}

func (m Macro) Expand(bound bindings) e.Expr {
	return walkAndReplace(m.Body, bound)
}

func (m Macro) ExpandHygienic(bound bindings, mark Mark) e.Expr {
	return walkAndReplaceHygienic(m.Body, bound, mark, m.PatternVars)
}

func walkAndReplaceHygienic(toWalk e.Expr, bound bindings, mark Mark, patternVars map[e.Identifier]bool) e.Expr {
	return walkAndReplaceHygienicImpl(toWalk, bound, mark, patternVars, nil)
}

// ExpandHygienicWithDefinitionBindings skips marking identifiers that were
// already bound at the macro's definition site, so references to built-ins
// and special forms resolve correctly after expansion.
func ExpandHygienicWithDefinitionBindings(body e.Expr, bound bindings, mark Mark, patternVars map[e.Identifier]bool, definitionBindings map[e.Identifier]bool) e.Expr {
	return walkAndReplaceHygienicImpl(body, bound, mark, patternVars, definitionBindings)
}

func walkAndReplaceHygienicImpl(toWalk e.Expr, bound bindings, mark Mark, patternVars map[e.Identifier]bool, definitionBindings map[e.Identifier]bool) e.Expr {
	if id, ok := toWalk.(e.Identifier); ok {
		replacement, present := bound[id]
		if present {
			return replacement
		}
		if definitionBindings != nil && definitionBindings[id] {
			return id
		}
		return e.ScopedIdentifier{
			Name:  id,
			Marks: map[uint64]bool{mark: true},
		}
	}
	if si, ok := toWalk.(e.ScopedIdentifier); ok {
		replacement, present := bound[si.Name]
		if present {
			return replacement
		}
		if definitionBindings != nil && definitionBindings[si.Name] {
			return si
		}
		newMarks := copyMarks(si.Marks)
		newMarks[mark] = true
		return e.ScopedIdentifier{
			Name:  si.Name,
			Marks: newMarks,
		}
	}

	if list, ok := toWalk.(e.List); ok && list != e.NIL {
		h := list.First()
		return e.Cons(
			walkAndReplaceHygienicImpl(h, bound, mark, patternVars, definitionBindings),
			walkAndReplaceHygienicImpl(list.Second(), bound, mark, patternVars, definitionBindings),
		)
	}

	return toWalk
}

func walkAndReplace(toWalk e.Expr, bound bindings) e.Expr {
	if id, ok := toWalk.(e.Identifier); ok {
		replacement, present := bound[id]
		if present {
			return replacement
		}
	}
	if si, ok := toWalk.(e.ScopedIdentifier); ok {
		replacement, present := bound[si.Name]
		if present {
			return replacement
		}
	}

	if list, ok := toWalk.(e.List); ok && list != e.NIL {
		h := list.First()
		return e.Cons(walkAndReplace(h, bound), walkAndReplace(list.Second(), bound))
	}

	return toWalk
}

func matchWalk(macro e.Expr, code e.Expr, bound bindings, hasElipsis bool) (bool, bindings) {
	if id, ok := macro.(e.Identifier); ok {
		return matchAndBindIdentifier(id, code, bound)
	}
	if si, ok := macro.(e.ScopedIdentifier); ok {
		return matchAndBindIdentifier(si.Name, code, bound)
	}
	if macroList, macroOk := macro.(e.List); macroOk {
		if codeList, codeOk := code.(e.List); codeOk {
			return matchHeadAndTail(macroList, codeList, bound, hasElipsis)
		} else {
			return matchFinalCodeExpression(macroList, code, bound, hasElipsis)
		}
	}
	if code != nil && macro != nil && code.Equiv(macro) {
		return true, bound
	}
	return false, nil
}

func matchAndBindIdentifier(id e.Identifier, code e.Expr, bound bindings) (bool, bindings) {
	b, present := bound[id]
	if present && !b.Equiv(code) {
		return false, bound
	}
	bound[id] = code
	return true, bound
}

func matchHeadAndTail(macroList e.List, codeList e.List, bound bindings, hasElipsis bool) (bool, bindings) {
	if macroList == e.NIL && codeList == e.NIL {
		return true, bound
	}
	macroLength, _ := listLength(macroList)
	if macroList == e.NIL || (codeList == e.NIL && (!hasElipsis || macroLength > 1)) {
		return false, nil
	}

	if id := toIdentifier(macroList.First()); id == e.Identifier("...") {
		return matchElipsis(macroList, macroLength, codeList, bound)
	} else {
		headMatch, bound := matchWalk(macroList.First(), codeList.First(), bound, hasElipsis)
		if headMatch {
			return matchWalk(macroList.Second(), codeList.Second(), bound, hasElipsis)
		}
	}
	return false, nil
}

func matchElipsis(macroList e.List, macroLength int, codeList e.List, bound bindings) (bool, bindings) {
	macHead := macroList.First()
	if macroList.Second() == e.NIL {
		return matchWalk(macHead, codeList, bound, true)
	}

	followingPatternCount := macroLength - 1
	bindToMacHead, rest := splitListAt(followingPatternCount, codeList)
	_, bound = matchWalk(macHead, bindToMacHead, bound, true)

	return matchWalk(macroList.Second(), rest, bound, true)
}

func matchFinalCodeExpression(macroList e.List, code e.Expr, bound bindings, hasElipsis bool) (bool, bindings) {
	if macroList != e.NIL && macroList.Second() == e.NIL {
		return matchWalk(macroList.First(), code, bound, hasElipsis)
	}
	if id := toIdentifier(macroList.First()); id == e.Identifier("...") {
		bound[id] = e.NIL
		return matchWalk(macroList.Second(), code, bound, hasElipsis)
	}
	return false, nil
}

func idAndRest(expr e.Expr) (e.Identifier, e.Expr, error) {
	identifier := expr
	if list, ok := expr.(e.List); ok {
		id := toIdentifier(list.First())
		if id == "" {
			return "", nil, fmt.Errorf("macro pattern must contain identifiers, got %T", list.First())
		}
		return id, list.Second(), nil
	}
	id := toIdentifier(identifier)
	if id == "" {
		return "", nil, fmt.Errorf("macro pattern must contain identifiers, got %T", identifier)
	}
	return id, nil, nil
}

func toIdentifier(expr e.Expr) e.Identifier {
	switch v := expr.(type) {
	case e.Identifier:
		return v
	case e.ScopedIdentifier:
		return v.Name
	default:
		return ""
	}
}

func splitListAt(endCount int, codeList e.List) (e.Expr, e.Expr) {

	begining := codeList
	splitPoint := begining
	var ending e.Expr
	ok := false
	codeLength, _ := listLength(codeList)
	splitIndex := codeLength - (endCount + 1)

	if splitIndex < 0 {
		return e.NIL, codeList
	}

	for i := 0; i < splitIndex; i++ {
		if sp, ok := splitPoint.Tail(); ok {
			splitPoint = sp
		} else {
			break
		}
	}
	ending, ok = splitPoint.Tail()
	if splitPoint != e.NIL && (ok || endCount == 1) {
		ending = splitPoint.Second()
		if pair, ok := splitPoint.(*e.Pair); ok {
			pair.T = e.NIL
		}
	}

	return begining, ending
}

func listLength(list e.List) (int, bool) {
	count := 0
	ok := false
	for list != e.NIL {
		count = count + 1
		list, ok = list.Tail()
		if !ok {
			return count + 1, false
		}
	}
	return count, true
}
