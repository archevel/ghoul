package macromancy

import (
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

func TestSyntaxObjectWrapsExprAndPreservesDatum(t *testing.T) {
	so := SyntaxObject{Datum: e.Integer(42), Marks: NewMarkSet()}

	if so.Datum != e.Integer(42) {
		t.Errorf("expected datum to be 42, got %s", so.Datum.Repr())
	}

	if so.Repr() != "42" {
		t.Errorf("expected Repr() to be '42', got '%s'", so.Repr())
	}
}

func TestSyntaxObjectEquivComparesDataForNonIdentifiers(t *testing.T) {
	so1 := SyntaxObject{Datum: e.Integer(42), Marks: NewMarkSet()}
	so2 := SyntaxObject{Datum: e.Integer(42), Marks: NewMarkSet()}
	so3 := SyntaxObject{Datum: e.Integer(99), Marks: NewMarkSet()}

	if !so1.Equiv(so2) {
		t.Error("expected equivalent syntax objects to be Equiv")
	}
	if so1.Equiv(so3) {
		t.Error("expected non-equivalent syntax objects to not be Equiv")
	}
}

func TestMarkSetToggle(t *testing.T) {
	ms := NewMarkSet()

	ms2 := ms.Toggle(1)
	if !ms2[1] {
		t.Error("expected mark 1 to be present after adding")
	}
	if !ms.IsEmpty() {
		t.Error("original mark set should be unchanged (immutable toggle)")
	}

	ms3 := ms2.Toggle(1)
	if !ms3.IsEmpty() {
		t.Error("expected mark 1 to be removed after toggling again")
	}

	ms4 := ms.Toggle(1).Toggle(2)
	if !ms4[1] || !ms4[2] {
		t.Error("expected both marks 1 and 2 to be present")
	}
}

func TestMarkSetEquals(t *testing.T) {
	a := NewMarkSet().Toggle(1).Toggle(2)
	b := NewMarkSet().Toggle(2).Toggle(1)
	c := NewMarkSet().Toggle(1)

	if !MarksEqual(a, b) {
		t.Error("expected equal mark sets to be equal")
	}
	if MarksEqual(a, c) {
		t.Error("expected different mark sets to not be equal")
	}
	if !MarksEqual(NewMarkSet(), NewMarkSet()) {
		t.Error("expected two empty mark sets to be equal")
	}
}

func TestSyntaxObjectIdentifierEquivCheckMarks(t *testing.T) {
	so1 := SyntaxObject{Datum: e.Identifier("x"), Marks: NewMarkSet().Toggle(1)}
	so2 := SyntaxObject{Datum: e.Identifier("x"), Marks: NewMarkSet().Toggle(2)}
	so3 := SyntaxObject{Datum: e.Identifier("x"), Marks: NewMarkSet().Toggle(1)}
	so4 := SyntaxObject{Datum: e.Identifier("y"), Marks: NewMarkSet().Toggle(1)}

	if so1.Equiv(so2) {
		t.Error("same name, different marks should NOT be Equiv")
	}
	if !so1.Equiv(so3) {
		t.Error("same name, same marks should be Equiv")
	}
	if so1.Equiv(so4) {
		t.Error("different name, same marks should NOT be Equiv")
	}
}

func TestSyntaxObjectIdentifierEquivEmptyMarksMatchesPlainIdentifier(t *testing.T) {
	so := SyntaxObject{Datum: e.Identifier("x"), Marks: NewMarkSet()}

	if !so.Equiv(e.Identifier("x")) {
		t.Error("SyntaxObject with empty marks should be Equiv to plain Identifier")
	}

	soWithMarks := SyntaxObject{Datum: e.Identifier("x"), Marks: NewMarkSet().Toggle(1)}
	if soWithMarks.Equiv(e.Identifier("x")) {
		t.Error("SyntaxObject with marks should NOT be Equiv to plain Identifier")
	}
}

func TestWrapExprWrapsLeafNodes(t *testing.T) {
	wrapped := WrapExpr(e.Integer(42), NewMarkSet())
	so, ok := wrapped.(SyntaxObject)
	if !ok {
		t.Fatalf("expected SyntaxObject, got %T", wrapped)
	}
	if so.Datum != e.Integer(42) {
		t.Errorf("expected datum 42, got %s", so.Datum.Repr())
	}
}

func TestWrapExprWrapsPairRecursively(t *testing.T) {
	pair := e.Cons(e.Identifier("a"), e.Cons(e.Integer(1), e.NIL))
	wrapped := WrapExpr(pair, NewMarkSet())

	list, ok := wrapped.(e.List)
	if !ok {
		t.Fatalf("expected wrapped pair to implement List, got %T", wrapped)
	}

	head := list.First()
	headSo, ok := head.(SyntaxObject)
	if !ok {
		t.Fatalf("expected head to be SyntaxObject, got %T", head)
	}
	if headSo.Datum != e.Identifier("a") {
		t.Errorf("expected head datum 'a', got %s", headSo.Datum.Repr())
	}

	tail, ok := list.Tail()
	if !ok {
		t.Fatal("expected tail to be a list")
	}
	second := tail.First()
	secondSo, ok := second.(SyntaxObject)
	if !ok {
		t.Fatalf("expected second to be SyntaxObject, got %T", second)
	}
	if secondSo.Datum != e.Integer(1) {
		t.Errorf("expected second datum 1, got %s", secondSo.Datum.Repr())
	}
}

func TestWrapExprPreservesNIL(t *testing.T) {
	wrapped := WrapExpr(e.NIL, NewMarkSet())
	if wrapped != e.NIL {
		t.Errorf("expected NIL to stay NIL, got %T", wrapped)
	}
}

func TestApplyMarkTogglesMarkOnIdentifiers(t *testing.T) {
	// Wrap a tree: (a 1)
	tree := WrapExpr(e.Cons(e.Identifier("a"), e.Cons(e.Integer(1), e.NIL)), NewMarkSet())

	marked := ApplyMark(tree, 5)

	list, ok := marked.(e.List)
	if !ok {
		t.Fatalf("expected List, got %T", marked)
	}

	headSo, ok := list.First().(SyntaxObject)
	if !ok {
		t.Fatalf("expected head to be SyntaxObject, got %T", list.First())
	}
	if !headSo.Marks[5] {
		t.Error("expected identifier 'a' to have mark 5")
	}

	// Non-identifier (integer) should not get marks
	tail, _ := list.Tail()
	secondSo, ok := tail.First().(SyntaxObject)
	if !ok {
		t.Fatalf("expected second to be SyntaxObject, got %T", tail.First())
	}
	if secondSo.Marks[5] {
		t.Error("expected non-identifier to NOT get mark 5")
	}
}

func TestApplyMarkTogglesExistingMark(t *testing.T) {
	// Identifier already has mark 5
	tree := SyntaxObject{Datum: e.Identifier("x"), Marks: NewMarkSet().Toggle(5)}
	result := ApplyMark(tree, 5)

	so, ok := result.(SyntaxObject)
	if !ok {
		t.Fatalf("expected SyntaxObject, got %T", result)
	}
	if !so.Marks.IsEmpty() {
		t.Error("expected mark 5 to be toggled off")
	}
}

func TestExtractPatternVars(t *testing.T) {
	// Pattern: (swap x y)
	pattern := e.Cons(e.Identifier("swap"), e.Cons(e.Identifier("x"), e.Cons(e.Identifier("y"), e.NIL)))

	vars := ExtractPatternVars(pattern)

	// swap is the macro name (first element), x and y are pattern variables
	if vars[e.Identifier("swap")] {
		t.Error("macro name 'swap' should NOT be a pattern variable")
	}
	if !vars[e.Identifier("x")] {
		t.Error("'x' should be a pattern variable")
	}
	if !vars[e.Identifier("y")] {
		t.Error("'y' should be a pattern variable")
	}
}

func TestExtractPatternVarsNested(t *testing.T) {
	// Pattern: (my-let ((x v)) body)
	pattern := e.Cons(e.Identifier("my-let"),
		e.Cons(e.Cons(e.Cons(e.Identifier("x"), e.Cons(e.Identifier("v"), e.NIL)), e.NIL),
			e.Cons(e.Identifier("body"), e.NIL)))

	vars := ExtractPatternVars(pattern)

	if vars[e.Identifier("my-let")] {
		t.Error("macro name should NOT be a pattern variable")
	}
	if !vars[e.Identifier("x")] || !vars[e.Identifier("v")] || !vars[e.Identifier("body")] {
		t.Error("x, v, body should all be pattern variables")
	}
}

func TestSyntaxObjectEquivWithPlainExpr(t *testing.T) {
	so := SyntaxObject{Datum: e.Integer(42), Marks: NewMarkSet()}

	if !so.Equiv(e.Integer(42)) {
		t.Error("expected SyntaxObject to be Equiv to its datum")
	}
}
