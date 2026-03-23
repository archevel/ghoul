package macromancy

import (
	e "github.com/archevel/ghoul/expressions"
)

// Mark is a unique identifier for a macro expansion invocation.
type Mark = uint64

// MarkSet tracks which marks apply to an identifier.
type MarkSet map[Mark]bool

func NewMarkSet() MarkSet {
	return MarkSet{}
}

// Toggle returns a new MarkSet with the given mark toggled (added if absent, removed if present).
func (ms MarkSet) Toggle(m Mark) MarkSet {
	result := MarkSet{}
	for k, v := range ms {
		result[k] = v
	}
	if result[m] {
		delete(result, m)
	} else {
		result[m] = true
	}
	return result
}

func (ms MarkSet) IsEmpty() bool {
	return len(ms) == 0
}

// WrapExpr recursively wraps leaf expressions in an expression tree as SyntaxObjects.
// Pairs are preserved as Pairs (so List interface works), but their elements are wrapped.
// NIL is preserved as-is.
func WrapExpr(expr e.Expr, marks MarkSet) e.Expr {
	if expr == e.NIL {
		return e.NIL
	}
	if pair, ok := expr.(*e.Pair); ok {
		return e.Cons(WrapExpr(pair.H, marks), WrapExpr(pair.T, marks))
	}
	if list, ok := expr.(e.List); ok && list != e.NIL {
		return e.Cons(WrapExpr(list.First(), marks), WrapExpr(list.Second(), marks))
	}
	return SyntaxObject{Datum: expr, Marks: copyMarks(marks)}
}

func copyMarks(ms MarkSet) MarkSet {
	result := MarkSet{}
	for k, v := range ms {
		result[k] = v
	}
	return result
}

// ApplyMark toggles a mark on all identifier SyntaxObjects in a tree.
func ApplyMark(expr e.Expr, mark Mark) e.Expr {
	if so, ok := expr.(SyntaxObject); ok {
		if _, isIdent := so.Datum.(e.Identifier); isIdent {
			return SyntaxObject{Datum: so.Datum, Marks: so.Marks.Toggle(mark)}
		}
		return so
	}
	if expr == e.NIL {
		return e.NIL
	}
	if pair, ok := expr.(*e.Pair); ok {
		return e.Cons(ApplyMark(pair.H, mark), ApplyMark(pair.T, mark))
	}
	return expr
}

// ExtractPatternVars extracts all identifiers from a macro pattern,
// excluding the first identifier (the macro name).
func ExtractPatternVars(pattern e.Expr) map[e.Identifier]bool {
	vars := map[e.Identifier]bool{}
	list, ok := pattern.(e.List)
	if !ok || list == e.NIL {
		return vars
	}
	// Skip the first element (macro name)
	rest := list.Second()
	collectIdentifiers(rest, vars)
	return vars
}

func collectIdentifiers(expr e.Expr, vars map[e.Identifier]bool) {
	if id, ok := expr.(e.Identifier); ok {
		if id != e.Identifier("...") {
			vars[id] = true
		}
		return
	}
	if si, ok := expr.(e.ScopedIdentifier); ok {
		if si.Name != e.Identifier("...") {
			vars[si.Name] = true
		}
		return
	}
	if expr == e.NIL {
		return
	}
	if list, ok := expr.(e.List); ok {
		collectIdentifiers(list.First(), vars)
		collectIdentifiers(list.Second(), vars)
	}
}

func MarksEqual(a, b MarkSet) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// SyntaxObject wraps an Expr with lexical context (marks) for hygiene.
type SyntaxObject struct {
	Datum e.Expr
	Marks MarkSet
}

func (so SyntaxObject) Repr() string {
	return so.Datum.Repr()
}

func (so SyntaxObject) Equiv(other e.Expr) bool {
	_, isIdent := so.Datum.(e.Identifier)

	if otherSo, ok := other.(SyntaxObject); ok {
		if isIdent {
			return so.Datum.Equiv(otherSo.Datum) && MarksEqual(so.Marks, otherSo.Marks)
		}
		return so.Datum.Equiv(otherSo.Datum)
	}

	if isIdent {
		return so.Marks.IsEmpty() && so.Datum.Equiv(other)
	}
	return so.Datum.Equiv(other)
}
