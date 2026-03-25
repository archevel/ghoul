package macromancy

// These tests exercise defensive code paths that are only reachable by
// constructing malformed internal state (improper lists, mismatched
// bindings, non-Pair List implementations, etc.). They exist purely
// for documentation and to verify that the code handles these edge
// cases gracefully rather than panicking.
//
// None of these scenarios are reachable from valid Ghoul source code.

import (
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

// --- matchFinalCodeExpression: `...` as head facing a non-list atom ---
// This requires `...` to be the first element of a pattern list when
// matchWalk dispatches to matchFinalCodeExpression (code is an atom).
// Normal patterns never produce this because `x ...` is intercepted
// by matchRepeatedEllipsis earlier. We construct it directly.
func TestMatchFinalCodeExpression_EllipsisHeadFacingAtom(t *testing.T) {
	// macroList = (... y), code = 42 (atom, not a list)
	macroList := e.Cons(e.Identifier("..."), e.Cons(e.Identifier("y"), e.NIL))
	bound := newBindings()

	ok, result := matchFinalCodeExpression(macroList, e.Integer(42), bound, false, nil)
	if !ok {
		t.Fatal("should match: ... binds to NIL (zero repetitions), y binds to 42")
	}
	if !result.vars[e.Identifier("y")].Equiv(e.Integer(42)) {
		t.Errorf("expected y=42, got %v", result.vars[e.Identifier("y")])
	}
	if !result.vars[e.Identifier("...")].Equiv(e.NIL) {
		t.Errorf("expected ...=NIL, got %v", result.vars[e.Identifier("...")])
	}
}

// --- bindingsForIteration: repeated var with out-of-range index ---
// This guard prevents a panic if repeated vars somehow have mismatched
// lengths. In practice matchRepeatedEllipsis always produces equal-length
// lists, so this is a defensive check against internal bugs.
func TestBindingsForIteration_OutOfRangeIndex(t *testing.T) {
	bound := newBindings()
	bound.repeated[e.Identifier("x")] = []e.Expr{e.Integer(1)} // length 1

	// Request index 5, which is out of range — should not panic
	iter := bindingsForIteration(bound, []e.Identifier{e.Identifier("x")}, 5)

	// x should not appear in iter.vars since the index was out of range
	if _, exists := iter.vars[e.Identifier("x")]; exists {
		t.Error("out-of-range index should not produce a binding")
	}
}

// --- bindingsForIteration: repeated var not in bound.repeated ---
// If the template references a variable as repeated but it was never
// captured, the lookup should silently skip it.
func TestBindingsForIteration_MissingRepeatedVar(t *testing.T) {
	bound := newBindings()

	iter := bindingsForIteration(bound, []e.Identifier{e.Identifier("ghost")}, 0)

	if _, exists := iter.vars[e.Identifier("ghost")]; exists {
		t.Error("missing repeated var should not produce a binding")
	}
}

// --- matchWalk: ScopedIdentifier in macro pattern ---
// When a macro expands to code containing define-syntax, pattern
// identifiers may carry marks from the outer expansion. matchWalk
// must handle ScopedIdentifier in the pattern position.
func TestMatchWalk_ScopedIdentifierPattern(t *testing.T) {
	macro := e.ScopedIdentifier{Name: "x", Marks: map[uint64]bool{1: true}}
	code := e.Integer(42)
	bound := newBindings()

	ok, result := matchWalk(macro, code, bound, false, nil)
	if !ok {
		t.Fatal("ScopedIdentifier pattern should match and bind by name")
	}
	if !result.vars[e.Identifier("x")].Equiv(e.Integer(42)) {
		t.Errorf("expected x=42, got %v", result.vars[e.Identifier("x")])
	}
}

// --- extractLiterals: improper list (dotted pair) in literals position ---
// A malformed syntax-rules form where the literals list is an improper
// list, e.g., (a . b) instead of (a b). The loop should stop at the
// non-list tail without panicking.
func TestExtractLiterals_ImproperList(t *testing.T) {
	// (a . b) — improper list, b is not a list
	improperList := e.Cons(e.Identifier("a"), e.Identifier("b"))

	literals, err := extractLiterals(improperList)
	if err != nil {
		t.Fatalf("should not error on improper list, got: %v", err)
	}
	if !literals[e.Identifier("a")] {
		t.Error("'a' should be extracted as a literal")
	}
	// 'b' is the dotted tail, not a list element — it should be ignored
	if literals[e.Identifier("b")] {
		t.Error("'b' (dotted tail) should not be extracted as a literal")
	}
}

// --- collectEllipsisVars: improper list in pattern ---
// A malformed pattern like (x . ...) where the tail is a dotted
// identifier. The Tail() call returns !ok, so the loop should break.
func TestCollectEllipsisVars_ImproperListPattern(t *testing.T) {
	// (x . y) — improper list as pattern body (after macro name stripped)
	improper := e.Cons(e.Identifier("x"), e.Identifier("y"))
	vars := map[e.Identifier]bool{}

	// Should not panic
	collectEllipsisVars(improper, vars, nil)

	// x is not followed by ..., so no ellipsis vars
	if len(vars) != 0 {
		t.Errorf("expected 0 ellipsis vars from improper list, got %d", len(vars))
	}
}

// --- collectEllipsisVars: `...` as dotted tail after subpattern ---
// Pattern like (x ... . z) where after skipping past `...`, the
// remaining Tail() call returns !ok because z is a dotted tail.
func TestCollectEllipsisVars_EllipsisFollowedByDottedTail(t *testing.T) {
	// Construct (x ... . z) — x followed by ... with z as dotted tail
	// This is: Cons(x, Cons(..., z))
	pattern := e.Cons(e.Identifier("x"), e.Cons(e.Identifier("..."), e.Identifier("z")))
	vars := map[e.Identifier]bool{}

	collectEllipsisVars(pattern, vars, nil)

	if !vars[e.Identifier("x")] {
		t.Error("x should be an ellipsis variable (precedes ...)")
	}
}

// --- matchRepeatedEllipsis: nil repeated map ---
// When the bindings struct is created with a zero-value (no
// repeated map initialized), the nil check at line 300 should
// initialize it rather than panicking.
func TestMatchRepeatedEllipsis_NilRepeatedMap(t *testing.T) {
	// Create bindings with nil repeated map (zero-value struct)
	bound := bindings{vars: map[e.Identifier]e.Expr{}}

	subPattern := e.Identifier("x")
	ellipsisAndRest := e.Cons(e.Identifier("..."), e.NIL)
	codeList := e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.NIL))

	ok, result := matchRepeatedEllipsis(subPattern, ellipsisAndRest, codeList, bound, nil)
	if !ok {
		t.Fatal("should match")
	}
	xVals := result.repeated[e.Identifier("x")]
	if len(xVals) != 2 {
		t.Fatalf("expected 2 x repetitions, got %d", len(xVals))
	}
}
