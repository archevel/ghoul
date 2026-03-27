package macromancy

// These tests exercise edge cases and defensive code paths in the
// Node-based pattern matching and expansion. Many of these scenarios
// can only be reached by constructing malformed internal state, not
// through valid Ghoul source code. They exist to document behavior
// and prevent panics.

import (
	"testing"

	"github.com/archevel/ghoul/bones"
)

// --- matchExpr: empty list matching ---

func TestMatchNodeExprNilMatchesNil(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.Nil, bones.Nil, bound, nil)
	if !ok {
		t.Error("Nil should match Nil")
	}
}

func TestMatchNodeExprEmptyListMatchesNil(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.NewListNode(nil), bones.Nil, bound, nil)
	if !ok {
		t.Error("empty ListNode should match Nil")
	}
}

func TestMatchNodeExprNilMatchesEmptyList(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.Nil, bones.NewListNode(nil), bound, nil)
	if !ok {
		t.Error("Nil should match empty ListNode")
	}
}

func TestMatchNodeExprNilDoesNotMatchNonEmpty(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.Nil, bones.IntNode(42), bound, nil)
	if ok {
		t.Error("Nil should not match non-empty value")
	}
}

// --- matchExpr: literal value matching ---

func TestMatchNodeExprIntegerEquiv(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.IntNode(42), bones.IntNode(42), bound, nil)
	if !ok {
		t.Error("matching integers should succeed")
	}
}

func TestMatchNodeExprIntegerMismatch(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.IntNode(1), bones.IntNode(2), bound, nil)
	if ok {
		t.Error("different integers should not match")
	}
}

func TestMatchNodeExprBooleanLiteral(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.BoolNode(true), bones.BoolNode(true), bound, nil)
	if !ok {
		t.Error("matching booleans should succeed")
	}
}

func TestMatchNodeExprStringLiteral(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.StrNode("hi"), bones.StrNode("hi"), bound, nil)
	if !ok {
		t.Error("matching strings should succeed")
	}
}

func TestMatchNodeExprTypeMismatch(t *testing.T) {
	bound := newBindings()
	ok, _ := matchExpr(bones.IntNode(1), bones.StrNode("1"), bound, nil)
	if ok {
		t.Error("int should not match string")
	}
}

// --- matchExpr: variable binding conflicts ---

func TestMatchNodeExprVariableAlreadyBoundSameValue(t *testing.T) {
	bound := newBindings()
	bound.vars["x"] = bones.IntNode(42)
	ok, _ := matchExpr(bones.IdentNode("x"), bones.IntNode(42), bound, nil)
	if !ok {
		t.Error("rebinding same value should succeed")
	}
}

func TestMatchNodeExprVariableAlreadyBoundDifferentValue(t *testing.T) {
	bound := newBindings()
	bound.vars["x"] = bones.IntNode(42)
	ok, _ := matchExpr(bones.IdentNode("x"), bones.IntNode(99), bound, nil)
	if ok {
		t.Error("rebinding different value should fail")
	}
}

// --- matchChildren: ellipsis edge cases ---

func TestMatchNodeChildrenEllipsisTailPatternsExceedCode(t *testing.T) {
	// Pattern: (mac x ... y z) with code (mac 1)
	// x... needs 0, but y and z need 2 — only 1 available
	pattern := []*bones.Node{n("x"), n("..."), n("y"), n("z")}
	code := []*bones.Node{i(1)}
	bound := newBindings()
	ok, _ := matchChildren(pattern, code, bound, nil)
	if ok {
		t.Error("should fail — not enough code elements for tail patterns")
	}
}

func TestMatchNodeChildrenEllipsisZeroRepetitionsWithTail(t *testing.T) {
	// Pattern: (mac x ... y) with code (mac 99)
	// x... gets 0 repetitions, y gets 99
	pattern := []*bones.Node{n("x"), n("..."), n("y")}
	code := []*bones.Node{i(99)}
	bound := newBindings()
	ok, result := matchChildren(pattern, code, bound, nil)
	if !ok {
		t.Fatal("should match with 0 repetitions + tail")
	}
	if len(result.repeated["x"]) != 0 {
		t.Errorf("expected 0 x repetitions, got %d", len(result.repeated["x"]))
	}
	if result.vars["y"].IntVal != 99 {
		t.Errorf("expected y=99, got %s", result.vars["y"].Repr())
	}
}

func TestMatchNodeChildrenSubpatternMismatchInEllipsis(t *testing.T) {
	// Pattern: (mac (a b) ...) with code (mac (1 2) 3)
	// Second code element 3 doesn't match subpattern (a b)
	m := Macro{
		Pattern: list(n("mac"), list(n("a"), n("b")), n("...")),
	}
	ok, _ := m.matches(list(n("mac"), list(i(1), i(2)), i(3)))
	if ok {
		t.Error("should fail — atom doesn't match list subpattern")
	}
}

func TestMatchNodeChildrenNilRepeatedMap(t *testing.T) {
	// Ensure that a zero-value bindings (nil repeated map) doesn't
	// panic when ellipsis initializes it.
	bound := bindings{vars: map[string]*bones.Node{}}
	// Pattern: x ..., code: 1 2
	pattern := []*bones.Node{n("x"), n("...")}
	code := []*bones.Node{i(1), i(2)}
	ok, result := matchChildren(pattern, code, bound, nil)
	if !ok {
		t.Fatal("should match")
	}
	if len(result.repeated["x"]) != 2 {
		t.Fatalf("expected 2 x repetitions, got %d", len(result.repeated["x"]))
	}
}

// --- matchExpr: list pattern vs non-list code ---

func TestMatchNodeExprListPatternVsAtom(t *testing.T) {
	// Pattern: (a b), Code: 42
	bound := newBindings()
	ok, _ := matchExpr(list(n("a"), n("b")), i(42), bound, nil)
	if ok {
		t.Error("list pattern should not match atom")
	}
}

func TestMatchNodeExprSingleElementListPatternVsAtom(t *testing.T) {
	// Pattern: (x), Code: 42 — single-element list pattern matches the atom
	// This handles improper list tail matching.
	bound := newBindings()
	ok, result := matchExpr(list(n("x")), i(42), bound, nil)
	if !ok {
		t.Fatal("single-element list pattern should match atom")
	}
	if result.vars["x"].IntVal != 42 {
		t.Errorf("expected x=42, got %s", result.vars["x"].Repr())
	}
}

// --- bindingsForIteration: edge cases ---

func TestNodeBindingsForIterationOutOfRange(t *testing.T) {
	bound := newBindings()
	bound.repeated["x"] = []*bones.Node{i(1)} // length 1

	// Index 5 is out of range — should not panic
	iter := bindingsForIteration(bound, []string{"x"}, 5)
	if _, exists := iter.vars["x"]; exists {
		t.Error("out-of-range index should not produce a binding")
	}
}

func TestNodeBindingsForIterationMissingVar(t *testing.T) {
	bound := newBindings()
	iter := bindingsForIteration(bound, []string{"ghost"}, 0)
	if _, exists := iter.vars["ghost"]; exists {
		t.Error("missing repeated var should not produce a binding")
	}
}

func TestNodeBindingsForIterationCopiesExistingVars(t *testing.T) {
	bound := newBindings()
	bound.vars["existing"] = i(42)
	bound.repeated["x"] = []*bones.Node{i(1), i(2)}

	iter := bindingsForIteration(bound, []string{"x"}, 0)
	if iter.vars["existing"].IntVal != 42 {
		t.Error("existing vars should be copied")
	}
	if iter.vars["x"].IntVal != 1 {
		t.Error("repeated var should be bound to i-th value")
	}
}

// --- expandHygienicImpl: edge cases ---

func TestExpandNodeHygienicNilInput(t *testing.T) {
	result := expandHygienic(nil, newBindings(), 1, nil, nil)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

func TestExpandNodeHygienicNilNode(t *testing.T) {
	result := expandHygienic(bones.Nil, newBindings(), 1, nil, nil)
	if !result.IsNil() {
		t.Error("Nil input should return Nil")
	}
}

func TestExpandNodeHygienicLiteralPassthrough(t *testing.T) {
	// Non-identifier, non-list nodes pass through unchanged
	result := expandHygienic(i(42), newBindings(), 1, nil, nil)
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestExpandNodeHygienicBooleanPassthrough(t *testing.T) {
	result := expandHygienic(bones.BoolNode(true), newBindings(), 1, nil, nil)
	if result.Kind != bones.BooleanNode || !result.BoolVal {
		t.Errorf("expected #t, got %s", result.Repr())
	}
}

func TestExpandNodeHygienicEmptyListBody(t *testing.T) {
	// A list with no children should produce an empty list
	result := expandHygienic(bones.NewListNode(nil), newBindings(), 1, nil, nil)
	if result.Kind != bones.ListNode {
		t.Errorf("expected ListNode, got %d", result.Kind)
	}
	if len(result.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(result.Children))
	}
}

// --- findRepeatedVars: edge cases ---

func TestFindNodeRepeatedVarsNonIdentifier(t *testing.T) {
	bound := newBindings()
	bound.repeated["x"] = []*bones.Node{i(1)}
	// Template is a literal — no repeated vars
	result := findRepeatedVars(i(42), bound)
	if len(result) != 0 {
		t.Errorf("expected 0 repeated vars from literal, got %d", len(result))
	}
}

func TestFindNodeRepeatedVarsNested(t *testing.T) {
	bound := newBindings()
	bound.repeated["x"] = []*bones.Node{i(1)}
	bound.repeated["y"] = []*bones.Node{i(2)}
	// Template: (+ x y) — both x and y are repeated
	result := findRepeatedVars(list(n("+"), n("x"), n("y")), bound)
	if len(result) != 2 {
		t.Errorf("expected 2 repeated vars, got %d", len(result))
	}
}

func TestFindNodeRepeatedVarsNoMatch(t *testing.T) {
	bound := newBindings()
	bound.repeated["x"] = []*bones.Node{i(1)}
	// Template: y — not a repeated var
	result := findRepeatedVars(n("y"), bound)
	if len(result) != 0 {
		t.Errorf("expected 0 repeated vars, got %d", len(result))
	}
}

// --- collectIdentifiers: edge cases ---

func TestCollectNodeIdentifiersSkipsEllipsis(t *testing.T) {
	vars := map[string]bool{}
	collectIdentifiers(n("..."), vars, nil)
	if vars["..."] {
		t.Error("... should not be collected as identifier")
	}
}

func TestCollectNodeIdentifiersSkipsWildcard(t *testing.T) {
	vars := map[string]bool{}
	collectIdentifiers(n("_"), vars, nil)
	if vars["_"] {
		t.Error("_ should not be collected as identifier")
	}
}

func TestCollectNodeIdentifiersSkipsLiterals(t *testing.T) {
	vars := map[string]bool{}
	literals := map[string]bool{"=>": true}
	collectIdentifiers(n("=>"), vars, literals)
	if vars["=>"] {
		t.Error("literals should not be collected as identifiers")
	}
}

func TestCollectNodeIdentifiersNonIdentifier(t *testing.T) {
	vars := map[string]bool{}
	collectIdentifiers(i(42), vars, nil)
	if len(vars) != 0 {
		t.Error("non-identifier should not produce vars")
	}
}

func TestCollectNodeIdentifiersRecursesIntoList(t *testing.T) {
	vars := map[string]bool{}
	collectIdentifiers(list(n("a"), list(n("b"), n("c"))), vars, nil)
	if !vars["a"] || !vars["b"] || !vars["c"] {
		t.Errorf("expected a, b, c in vars, got %v", vars)
	}
}

// --- extractLiterals: edge cases ---

func TestExtractNodeLiteralsNilNode(t *testing.T) {
	lits, err := extractLiterals(bones.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(lits) != 0 {
		t.Errorf("expected 0 literals from Nil, got %d", len(lits))
	}
}

func TestExtractNodeLiteralsNonList(t *testing.T) {
	_, err := extractLiterals(i(42))
	if err == nil {
		t.Error("expected error for non-list literals")
	}
}

func TestExtractNodeLiteralsNonIdentifierChild(t *testing.T) {
	_, err := extractLiterals(list(i(42)))
	if err == nil {
		t.Error("expected error for non-identifier in literals list")
	}
}

// --- extractMacros: edge cases ---

func TestExtractMacrosNonList(t *testing.T) {
	_, err := extractMacros("test", i(42))
	if err == nil {
		t.Error("expected error for non-list syntax-rules")
	}
}

func TestExtractMacrosTooFewChildren(t *testing.T) {
	_, err := extractMacros("test", list(n("syntax-rules"), list()))
	if err == nil {
		t.Error("expected error for syntax-rules with no rules")
	}
}

func TestExtractMacrosRuleNotList(t *testing.T) {
	// Rule is an atom, not a list
	_, err := extractMacros("test", list(n("syntax-rules"), list(), i(42)))
	if err == nil {
		t.Error("expected error for non-list rule")
	}
}

func TestExtractMacrosRuleTooShort(t *testing.T) {
	// Rule is a list with only one element (no body)
	_, err := extractMacros("test", list(n("syntax-rules"), list(), list(list(n("mac")))))
	if err == nil {
		t.Error("expected error for rule with no body")
	}
}

// --- isEmptyList: edge cases ---

func TestIsEmptyListNil(t *testing.T) {
	if !isEmptyList(bones.Nil) {
		t.Error("Nil should be empty list")
	}
}

func TestIsEmptyListEmptyListNode(t *testing.T) {
	if !isEmptyList(bones.NewListNode(nil)) {
		t.Error("empty ListNode should be empty list")
	}
}

func TestIsEmptyListNonEmpty(t *testing.T) {
	if isEmptyList(list(i(1))) {
		t.Error("non-empty ListNode should not be empty list")
	}
}

func TestIsEmptyListAtom(t *testing.T) {
	if isEmptyList(i(42)) {
		t.Error("atom should not be empty list")
	}
}

// --- isEllipsis ---

func TestIsNodeEllipsisTrue(t *testing.T) {
	if !isEllipsis(n("...")) {
		t.Error("... should be ellipsis")
	}
}

func TestIsNodeEllipsisFalse(t *testing.T) {
	if isEllipsis(n("x")) {
		t.Error("x should not be ellipsis")
	}
}

func TestIsNodeEllipsisNonIdentifier(t *testing.T) {
	if isEllipsis(i(42)) {
		t.Error("integer should not be ellipsis")
	}
}

// --- WrapSyntax / ApplyMark / ResolveSyntax: edge cases ---

func TestWrapSyntaxDottedTail(t *testing.T) {
	node := &bones.Node{Kind: bones.ListNode, Children: []*bones.Node{i(1)}, DottedTail: i(2)}
	wrapped := WrapSyntax(node, MarkSet{1: true})
	if wrapped.DottedTail == nil {
		t.Fatal("dotted tail should be wrapped")
	}
	if wrapped.DottedTail.Kind != bones.SyntaxObjectNode {
		t.Error("dotted tail should be SyntaxObjectNode")
	}
}

func TestApplyMarkDottedTail(t *testing.T) {
	node := &bones.Node{Kind: bones.ListNode, Children: []*bones.Node{bones.IdentNode("x")}, DottedTail: bones.IdentNode("y")}
	marked := ApplyMark(node, 1)
	if marked.DottedTail == nil || !marked.DottedTail.Marks[1] {
		t.Error("dotted tail identifier should get mark")
	}
}

func TestResolveSyntaxDottedTail(t *testing.T) {
	so := &bones.Node{Kind: bones.SyntaxObjectNode, Quoted: bones.IdentNode("x"), Marks: map[uint64]bool{1: true}}
	node := &bones.Node{Kind: bones.ListNode, Children: []*bones.Node{i(1)}, DottedTail: so}
	resolved := ResolveSyntax(node)
	if resolved.DottedTail == nil || resolved.DottedTail.Kind != bones.IdentifierNode {
		t.Error("dotted tail SyntaxObject should be resolved to identifier")
	}
	if !resolved.DottedTail.Marks[1] {
		t.Error("resolved dotted tail should carry marks")
	}
}

func TestApplyMarkNilPassthrough(t *testing.T) {
	result := ApplyMark(nil, 1)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

func TestApplyMarkNonIdentSyntaxObject(t *testing.T) {
	// SyntaxObject wrapping a non-identifier should pass through unchanged
	so := &bones.Node{Kind: bones.SyntaxObjectNode, Quoted: i(42), Marks: map[uint64]bool{}}
	result := ApplyMark(so, 1)
	if result != so {
		t.Error("non-identifier SyntaxObject should pass through")
	}
}

func TestResolveSyntaxNonIdentSyntaxObject(t *testing.T) {
	so := &bones.Node{Kind: bones.SyntaxObjectNode, Quoted: i(42)}
	resolved := ResolveSyntax(so)
	if resolved.Kind != bones.IntegerNode || resolved.IntVal != 42 {
		t.Errorf("expected unwrapped 42, got %s", resolved.Repr())
	}
}

func TestResolveSyntaxNilQuoted(t *testing.T) {
	so := &bones.Node{Kind: bones.SyntaxObjectNode, Quoted: nil}
	resolved := ResolveSyntax(so)
	if !resolved.IsNil() {
		t.Errorf("SyntaxObject with nil Quoted should resolve to Nil, got %s", resolved.Repr())
	}
}

func TestResolveSyntaxNonSyntaxPassthrough(t *testing.T) {
	node := i(42)
	resolved := ResolveSyntax(node)
	if resolved != node {
		t.Error("non-SyntaxObject non-list should pass through")
	}
}
