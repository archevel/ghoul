package macromancy

import (
	e "github.com/archevel/ghoul/expressions"
)

type Mark = uint64

type MarkSet map[Mark]bool

func NewMarkSet() MarkSet {
	return MarkSet{}
}

// Toggle returns a new MarkSet with the given mark flipped.
// This implements Racket's anti-mark behavior: applying the same
// mark twice cancels it out, which is how input expressions shed
// the macro-introduction mark while template expressions keep it.
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

// WrapExpr wraps leaf expressions as SyntaxObjects while preserving the
// Pair tree structure, so the List interface continues to work for traversal.
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

func ApplyMark(expr e.Expr, mark Mark) e.Expr {
	if so, ok := expr.(SyntaxObject); ok {
		if _, isIdent := so.Datum.(e.Identifier); isIdent {
			return SyntaxObject{Datum: so.Datum, Marks: so.Marks.Toggle(mark)}
		}
		return so
	}
	// Plain identifiers in the output came from the transformer itself
	// (not from the input, which would be wrapped in SyntaxObject).
	if id, ok := expr.(e.Identifier); ok {
		return e.ScopedIdentifier{Name: id, Marks: map[uint64]bool{mark: true}}
	}
	if si, ok := expr.(e.ScopedIdentifier); ok {
		return e.ScopedIdentifier{Name: si.Name, Marks: MarkSet(si.Marks).Toggle(mark)}
	}
	if expr == e.NIL {
		return e.NIL
	}
	if pair, ok := expr.(*e.Pair); ok {
		return e.Cons(ApplyMark(pair.H, mark), ApplyMark(pair.T, mark))
	}
	return expr
}

// ExtractPatternVars returns all identifiers in a macro pattern except
// the first one, which is the macro name rather than a variable to bind.
func ExtractPatternVars(pattern e.Expr) map[e.Identifier]bool {
	vars := map[e.Identifier]bool{}
	list, ok := pattern.(e.List)
	if !ok || list == e.NIL {
		return vars
	}
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

// ResolveExpr strips SyntaxObject wrappers, converting marked identifiers
// to ScopedIdentifier and unmarked ones back to plain Identifier.
func ResolveExpr(expr e.Expr) e.Expr {
	if so, ok := expr.(SyntaxObject); ok {
		if id, isIdent := so.Datum.(e.Identifier); isIdent {
			if so.Marks.IsEmpty() {
				return id
			}
			return e.ScopedIdentifier{Name: id, Marks: so.Marks}
		}
		return so.Datum
	}
	if expr == e.NIL {
		return e.NIL
	}
	if pair, ok := expr.(*e.Pair); ok {
		return e.Cons(ResolveExpr(pair.H), ResolveExpr(pair.T))
	}
	return expr
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
