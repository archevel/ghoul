package macromancy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
)

// SyntaxTransformer holds a pattern-based macro transformer (syntax-rules).
// Created by BuildSyntaxRulesTransformer and used by the expander to
// expand macro calls during the expansion phase.
type SyntaxTransformer struct {
	Transform func(code e.List, mark Mark) (e.Expr, error)
}

func (st SyntaxTransformer) Repr() string {
	return "#<syntax-transformer>"
}

func (st SyntaxTransformer) Equiv(other e.Expr) bool {
	return false
}

// BuildSyntaxRulesTransformer creates a SyntaxTransformer from a syntax-rules
// form. It parses the pattern/template pairs, and uses the provided set of
// definition-time bound identifiers for hygiene (identifiers in the set are
// not marked during expansion).
func BuildSyntaxRulesTransformer(name e.Identifier, syntaxRules e.List, definitionBindings map[e.Identifier]bool) (SyntaxTransformer, error) {
	defineSyntaxForm := e.Cons(e.Identifier("define-syntax"),
		e.Cons(name, e.Cons(syntaxRules, e.NIL)))

	mg, err := NewMacroGroup(defineSyntaxForm)
	if err != nil {
		return SyntaxTransformer{}, err
	}

	macros := mg.Macros()

	return SyntaxTransformer{
		Transform: func(code e.List, mark Mark) (e.Expr, error) {
			for _, m := range macros {
				if ok, bound := m.Matches(code); ok {
					return ExpandHygienicWithDefinitionBindings(m.Body, bound, mark, m.PatternVars, definitionBindings), nil
				}
			}
			return nil, fmt.Errorf("no matching pattern for %s", code.Repr())
		},
	}, nil
}

type bindings struct {
	vars     map[e.Identifier]e.Expr
	repeated map[e.Identifier][]e.Expr
}

func newBindings() bindings {
	return bindings{vars: map[e.Identifier]e.Expr{}, repeated: map[e.Identifier][]e.Expr{}}
}

type Macro struct {
	Pattern      e.Expr
	Body         e.Expr
	PatternVars  map[e.Identifier]bool
	EllipsisVars map[e.Identifier]bool
	Literals     map[e.Identifier]bool
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

// MatchAndBind matches the given expression against this macro's pattern.
// Returns (true, assocList) on success or (false, nil) on failure.
// The association list contains (name . value) pairs for each bound variable.
func (m Macro) MatchAndBind(expr e.Expr) (bool, e.Expr) {
	ok, bound := m.Matches(expr)
	if !ok {
		return false, nil
	}

	var result e.Expr = e.NIL
	for id, val := range bound.vars {
		result = e.Cons(e.Cons(id, val), result)
	}
	for id, vals := range bound.repeated {
		var valList e.Expr = e.NIL
		for i := len(vals) - 1; i >= 0; i-- {
			valList = e.Cons(vals[i], valList)
		}
		result = e.Cons(e.Cons(id, valList), result)
	}
	return true, result
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
		rest := list.Second()

		// When the head is `...`, splice its bound value into the parent list
		// rather than nesting it, so (begin x ...) becomes (begin a b c) not (begin a (b c))
		if isEllipsisIdentifier(h) {
			ellipsisBinding := lookupEllipsisBinding(bound)
			if ellipsisBinding != nil {
				return appendExprs(ellipsisBinding, walkAndReplaceHygienicImpl(rest, bound, mark, patternVars, definitionBindings))
			}
		}

		// Check if the next element is `...` and the head references repeated bindings.
		// If so, iterate through the repeated bindings and splice the results.
		if restList, restOk := rest.(e.List); restOk && restList != e.NIL && isEllipsisIdentifier(restList.First()) {
			repeatedVars := findRepeatedVarsInTemplate(h, bound)
			if len(repeatedVars) > 0 {
				afterEllipsis := restList.Second()
				expandedTail := walkAndReplaceHygienicImpl(afterEllipsis, bound, mark, patternVars, definitionBindings)

				count := len(bound.repeated[repeatedVars[0]])
				for i := count - 1; i >= 0; i-- {
					iterBound := bindingsForIteration(bound, repeatedVars, i)
					expanded := walkAndReplaceHygienicImpl(h, iterBound, mark, patternVars, definitionBindings)
					expandedTail = e.Cons(expanded, expandedTail)
				}
				return expandedTail
			}
		}

		return e.Cons(
			walkAndReplaceHygienicImpl(h, bound, mark, patternVars, definitionBindings),
			walkAndReplaceHygienicImpl(rest, bound, mark, patternVars, definitionBindings),
		)
	}

	return toWalk
}

func findRepeatedVarsInTemplate(tmpl e.Expr, bound bindings) []e.Identifier {
	var result []e.Identifier
	findRepeatedVarsWalk(tmpl, bound, &result)
	return result
}

func findRepeatedVarsWalk(expr e.Expr, bound bindings, result *[]e.Identifier) {
	if id, ok := expr.(e.Identifier); ok {
		if _, hasRepeated := bound.repeated[id]; hasRepeated {
			*result = append(*result, id)
		}
		return
	}
	if si, ok := expr.(e.ScopedIdentifier); ok {
		if _, hasRepeated := bound.repeated[si.Name]; hasRepeated {
			*result = append(*result, si.Name)
		}
		return
	}
	if list, ok := expr.(e.List); ok && list != e.NIL {
		findRepeatedVarsWalk(list.First(), bound, result)
		findRepeatedVarsWalk(list.Second(), bound, result)
	}
}

// bindingsForIteration creates a new bindings where each repeated variable
// is bound to its i-th value as a single var.
func bindingsForIteration(bound bindings, repeatedVars []e.Identifier, i int) bindings {
	iter := newBindings()
	for k, v := range bound.vars {
		iter.vars[k] = v
	}
	for _, v := range repeatedVars {
		if vals, ok := bound.repeated[v]; ok && i < len(vals) {
			iter.vars[v] = vals[i]
		}
	}
	// Copy repeated bindings for any nested ellipsis
	for k, v := range bound.repeated {
		iter.repeated[k] = v
	}
	return iter
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
	// Wildcard matches anything without creating a binding
	if id == e.Identifier("_") {
		return true, bound
	}
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

	if id := toIdentifier(macroList.First()); macroList != e.NIL && id == e.Identifier("...") {
		return matchEllipsis(macroList, macroLength, codeList, bound, literals)
	}

	// Check if the element after head is `...` — if so, use repeated matching.
	// This must be checked before the NIL-codeList bail-out, since zero
	// repetitions are valid for ellipsis patterns.
	if macroList != e.NIL {
		macroTail, tailOk := macroList.Tail()
		if tailOk && macroTail != e.NIL {
			if nextId := toIdentifier(macroTail.First()); nextId == e.Identifier("...") {
				return matchRepeatedEllipsis(macroList.First(), macroTail, codeList, bound, literals)
			}
		}
	}

	if macroList == e.NIL || (codeList == e.NIL && (!hasEllipsis || macroLength > 1)) {
		return false, bindings{}
	}

	headMatch, bound := matchWalk(macroList.First(), codeList.First(), bound, hasEllipsis, literals)
	if headMatch {
		return matchWalk(macroList.Second(), codeList.Second(), bound, hasEllipsis, literals)
	}
	return false, bindings{}
}

// matchRepeatedEllipsis handles `<subpattern> ...` by matching each code
// element against the subpattern and collecting per-variable repeated bindings.
func matchRepeatedEllipsis(subPattern e.Expr, ellipsisAndRest e.List, codeList e.List, bound bindings, literals map[e.Identifier]bool) (bool, bindings) {
	// Determine how many patterns follow the `...`
	afterEllipsis, afterOk := ellipsisAndRest.Tail()
	tailPatternCount := 0
	if afterOk && afterEllipsis != e.NIL {
		tailPatternCount = listLength(afterEllipsis)
	}

	// Split code: repeated portion vs tail
	codeLength := listLength(codeList)
	repeatedCount := codeLength - tailPatternCount
	if repeatedCount < 0 {
		repeatedCount = 0
	}

	// Collect identifiers in the subpattern to initialize repeated bindings
	subVars := map[e.Identifier]bool{}
	collectIdentifiers(subPattern, subVars, literals)
	for v := range subVars {
		if bound.repeated == nil {
			bound.repeated = map[e.Identifier][]e.Expr{}
		}
		bound.repeated[v] = []e.Expr{}
	}

	// Match each element against the subpattern
	current := codeList
	for i := 0; i < repeatedCount && current != e.NIL; i++ {
		localBound := newBindings()
		ok, localBound := matchWalk(subPattern, current.First(), localBound, false, literals)
		if !ok {
			return false, bindings{}
		}
		for v := range subVars {
			if val, exists := localBound.vars[v]; exists {
				bound.repeated[v] = append(bound.repeated[v], val)
			}
		}
		next, nextOk := current.Tail()
		if !nextOk {
			break
		}
		current = next
	}

	// Match remaining patterns after `...`
	if !afterOk || afterEllipsis == e.NIL {
		return true, bound
	}

	// Build the remaining code list (tail portion)
	var tailCode e.Expr = e.NIL
	if repeatedCount < codeLength {
		_, tailCode = splitListAt(tailPatternCount, codeList)
	}

	return matchWalk(afterEllipsis, tailCode, bound, false, literals)
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

// matchFinalCodeExpression handles the case where the code is not a list
// (e.g., the atom tail of a dotted pair) but the pattern still has elements.
// If only one pattern element remains, it matches against the atom.
// If the head is `...`, it binds to NIL (zero repetitions) and continues.
// Otherwise the pattern expects more structure than the code provides.
func matchFinalCodeExpression(macroList e.List, code e.Expr, bound bindings, hasEllipsis bool, literals map[e.Identifier]bool) (bool, bindings) {
	if macroList != e.NIL && macroList.Second() == e.NIL {
		return matchWalk(macroList.First(), code, bound, hasEllipsis, literals)
	}
	// When `...` faces a non-list code atom (e.g., from an improper list
	// where the dotted tail has been consumed), treat it as zero repetitions.
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
