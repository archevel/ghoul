package macromancy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
)

type bindings struct {
	vars     map[e.Identifier]e.Expr
	repeated map[e.Identifier][]e.Expr
}

func newBindings() bindings {
	return bindings{vars: map[e.Identifier]e.Expr{}, repeated: map[e.Identifier][]e.Expr{}}
}

type Macro struct {
	Pattern     e.Expr
	Body        e.Expr
	PatternVars map[e.Identifier]bool
	Literals    map[e.Identifier]bool
}

func (m Macro) Matches(expr e.Expr) (bool, bindings) {
	macId, macPat, err := idAndRest(m.Pattern)
	if err != nil {
		return false, bindings{}
	}
	codeId, code, err := idAndRest(expr)
	if err != nil {
		return false, bindings{}
	}
	if macId.Equiv(codeId) {
		if macPat == nil && code == nil {
			return true, newBindings()
		}
		return matchWalk(macPat, code, newBindings(), false, m.Literals)
	}
	return false, bindings{}
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
		replacement, present := bound.vars[id]
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
		replacement, present := bound.vars[si.Name]
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
		// When the head is `...`, splice its bound value into the parent list
		// rather than nesting it, so (begin x ...) becomes (begin a b c) not (begin a (b c))
		if isEllipsisIdentifier(h) {
			ellipsisBinding := lookupEllipsisBinding(bound)
			if ellipsisBinding != nil {
				return appendExprs(ellipsisBinding, walkAndReplaceHygienicImpl(list.Second(), bound, mark, patternVars, definitionBindings))
			}
		}
		return e.Cons(
			walkAndReplaceHygienicImpl(h, bound, mark, patternVars, definitionBindings),
			walkAndReplaceHygienicImpl(list.Second(), bound, mark, patternVars, definitionBindings),
		)
	}

	return toWalk
}

func isEllipsisIdentifier(expr e.Expr) bool {
	if id, ok := expr.(e.Identifier); ok {
		return id == e.Identifier("...")
	}
	if si, ok := expr.(e.ScopedIdentifier); ok {
		return si.Name == e.Identifier("...")
	}
	return false
}

func lookupEllipsisBinding(bound bindings) e.Expr {
	if val, ok := bound.vars[e.Identifier("...")]; ok {
		return val
	}
	return nil
}

// appendExprs appends the elements of a list to a tail expression,
// producing a proper list splice.
func appendExprs(list e.Expr, tail e.Expr) e.Expr {
	if list == e.NIL {
		return tail
	}
	if l, ok := list.(e.List); ok && l != e.NIL {
		return e.Cons(l.First(), appendExprs(l.Second(), tail))
	}
	return e.Cons(list, tail)
}

func matchWalk(macro e.Expr, code e.Expr, bound bindings, hasEllipsis bool, literals map[e.Identifier]bool) (bool, bindings) {
	if id, ok := macro.(e.Identifier); ok {
		return matchAndBindIdentifier(id, code, bound, literals)
	}
	if si, ok := macro.(e.ScopedIdentifier); ok {
		return matchAndBindIdentifier(si.Name, code, bound, literals)
	}
	if macroList, macroOk := macro.(e.List); macroOk {
		if codeList, codeOk := code.(e.List); codeOk {
			return matchHeadAndTail(macroList, codeList, bound, hasEllipsis, literals)
		} else {
			return matchFinalCodeExpression(macroList, code, bound, hasEllipsis, literals)
		}
	}
	if code != nil && macro != nil && code.Equiv(macro) {
		return true, bound
	}
	return false, bindings{}
}

func matchAndBindIdentifier(id e.Identifier, code e.Expr, bound bindings, literals map[e.Identifier]bool) (bool, bindings) {
	if literals != nil && literals[id] {
		codeId := toIdentifier(code)
		if codeId == id {
			return true, bound
		}
		return false, bindings{}
	}
	b, present := bound.vars[id]
	if present && !b.Equiv(code) {
		return false, bound
	}
	bound.vars[id] = code
	return true, bound
}

func matchHeadAndTail(macroList e.List, codeList e.List, bound bindings, hasEllipsis bool, literals map[e.Identifier]bool) (bool, bindings) {
	if macroList == e.NIL && codeList == e.NIL {
		return true, bound
	}
	macroLength := listLength(macroList)
	if macroList == e.NIL || (codeList == e.NIL && (!hasEllipsis || macroLength > 1)) {
		return false, bindings{}
	}

	if id := toIdentifier(macroList.First()); id == e.Identifier("...") {
		return matchEllipsis(macroList, macroLength, codeList, bound, literals)
	} else {
		headMatch, bound := matchWalk(macroList.First(), codeList.First(), bound, hasEllipsis, literals)
		if headMatch {
			return matchWalk(macroList.Second(), codeList.Second(), bound, hasEllipsis, literals)
		}
	}
	return false, bindings{}
}

func matchEllipsis(macroList e.List, macroLength int, codeList e.List, bound bindings, literals map[e.Identifier]bool) (bool, bindings) {
	macHead := macroList.First()
	if macroList.Second() == e.NIL {
		return matchWalk(macHead, codeList, bound, true, literals)
	}

	followingPatternCount := macroLength - 1
	bindToMacHead, rest := splitListAt(followingPatternCount, codeList)
	_, bound = matchWalk(macHead, bindToMacHead, bound, true, literals)

	return matchWalk(macroList.Second(), rest, bound, true, literals)
}

func matchFinalCodeExpression(macroList e.List, code e.Expr, bound bindings, hasEllipsis bool, literals map[e.Identifier]bool) (bool, bindings) {
	if macroList != e.NIL && macroList.Second() == e.NIL {
		return matchWalk(macroList.First(), code, bound, hasEllipsis, literals)
	}
	if id := toIdentifier(macroList.First()); id == e.Identifier("...") {
		bound.vars[id] = e.NIL
		return matchWalk(macroList.Second(), code, bound, hasEllipsis, literals)
	}
	return false, bindings{}
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

// splitListAt builds fresh lists for both halves rather than severing
// the original, since the same expression tree may be matched against
// multiple macro patterns.
func splitListAt(endCount int, codeList e.List) (e.Expr, e.Expr) {
	codeLength := listLength(codeList)
	beginCount := codeLength - endCount

	if beginCount <= 0 {
		return e.NIL, codeList
	}

	// Collect all elements (and possibly a dotted tail) into a slice
	type elem struct {
		val    e.Expr
		isTail bool // true if this is a non-list tail (dotted pair)
	}
	var elems []elem
	current := codeList
	for current != e.NIL {
		elems = append(elems, elem{val: current.First()})
		next, ok := current.Tail()
		if !ok {
			// Improper list: the Second() is the dotted tail
			elems = append(elems, elem{val: current.Second(), isTail: true})
			break
		}
		current = next
	}

	if beginCount > len(elems) {
		beginCount = len(elems)
	}

	// Build beginning from the first beginCount elements
	begElems := elems[:beginCount]
	endElems := elems[beginCount:]

	var beginning e.Expr = e.NIL
	for i := len(begElems) - 1; i >= 0; i-- {
		beginning = e.Cons(begElems[i].val, beginning)
	}

	// Build ending from the remaining elements
	if len(endElems) == 0 {
		return beginning, e.NIL
	}

	var ending e.Expr = e.NIL
	for i := len(endElems) - 1; i >= 0; i-- {
		if endElems[i].isTail {
			ending = endElems[i].val
		} else {
			ending = e.Cons(endElems[i].val, ending)
		}
	}

	return beginning, ending
}

func listLength(list e.List) int {
	count := 0
	for list != e.NIL {
		count++
		next, ok := list.Tail()
		if !ok {
			return count + 1
		}
		list = next
	}
	return count
}
