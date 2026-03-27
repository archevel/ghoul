package tome

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

// --- stripMarks ---

func TestStripNodeMarksOnList(t *testing.T) {
	// List containing scoped identifiers should have marks stripped
	list := e.NewListNode([]*e.Node{
		e.ScopedIdentNode("x", map[uint64]bool{1: true}),
		e.ScopedIdentNode("y", map[uint64]bool{2: true, 3: true}),
		e.IntNode(42),
	})
	result := stripMarks(list)
	if result.Kind != e.ListNode {
		t.Fatalf("expected ListNode, got %d", result.Kind)
	}
	if len(result.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(result.Children))
	}
	// First child: identifier with marks stripped
	if result.Children[0].Kind != e.IdentifierNode || result.Children[0].Name != "x" {
		t.Errorf("expected plain identifier 'x', got %s", result.Children[0].Repr())
	}
	if len(result.Children[0].Marks) > 0 {
		t.Error("marks should be stripped from first child")
	}
	// Second child: identifier with marks stripped
	if result.Children[1].Kind != e.IdentifierNode || result.Children[1].Name != "y" {
		t.Errorf("expected plain identifier 'y', got %s", result.Children[1].Repr())
	}
	if len(result.Children[1].Marks) > 0 {
		t.Error("marks should be stripped from second child")
	}
	// Third child: integer unchanged
	if result.Children[2].IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Children[2].Repr())
	}
}

func TestStripNodeMarksOnSyntaxObject(t *testing.T) {
	inner := e.IdentNode("foo")
	syntaxObj := &e.Node{Kind: e.SyntaxObjectNode, Quoted: inner}
	result := stripMarks(syntaxObj)
	if result.Kind != e.IdentifierNode || result.Name != "foo" {
		t.Errorf("expected identifier 'foo', got %s", result.Repr())
	}
}

func TestStripNodeMarksOnSyntaxObjectNilQuoted(t *testing.T) {
	syntaxObj := &e.Node{Kind: e.SyntaxObjectNode, Quoted: nil}
	result := stripMarks(syntaxObj)
	if !result.IsNil() {
		t.Errorf("expected Nil for SyntaxObject with nil quoted, got %s", result.Repr())
	}
}

func TestStripNodeMarksOnNil(t *testing.T) {
	result := stripMarks(e.Nil)
	if !result.IsNil() {
		t.Errorf("expected Nil, got %s", result.Repr())
	}
}

func TestStripNodeMarksOnNilPtr(t *testing.T) {
	result := stripMarks(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestStripNodeMarksOnAtom(t *testing.T) {
	// Non-identifier atom should pass through unchanged
	node := e.IntNode(99)
	result := stripMarks(node)
	if result != node {
		t.Error("atom should pass through unchanged")
	}
}

func TestStripNodeMarksOnPlainIdentifier(t *testing.T) {
	// Identifier without marks should pass through unchanged
	node := e.IdentNode("plain")
	result := stripMarks(node)
	if result != node {
		t.Error("plain identifier should pass through unchanged (same pointer)")
	}
}

// --- car/cdr/length error paths ---

func TestCarOnNilErrors(t *testing.T) {
	_, err := callStdlibDirect("car", []*e.Node{e.Nil})
	if err == nil {
		t.Error("expected error for car on empty list")
	}
}

func TestCarOnNonListErrors(t *testing.T) {
	_, err := callStdlibDirect("car", []*e.Node{e.IntNode(1)})
	if err == nil {
		t.Error("expected error for car on integer")
	}
}

func TestCdrOnNilErrors(t *testing.T) {
	_, err := callStdlibDirect("cdr", []*e.Node{e.Nil})
	if err == nil {
		t.Error("expected error for cdr on empty list")
	}
}

func TestCdrOnNonListErrors(t *testing.T) {
	_, err := callStdlibDirect("cdr", []*e.Node{e.IntNode(1)})
	if err == nil {
		t.Error("expected error for cdr on integer")
	}
}

func TestLengthOnNonListErrors(t *testing.T) {
	_, err := callStdlibDirect("length", []*e.Node{e.StrNode("hello")})
	if err == nil {
		t.Error("expected error for length on string")
	}
}

func TestLengthOnNilReturnsZero(t *testing.T) {
	result, err := callStdlibDirect("length", []*e.Node{e.Nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IntVal != 0 {
		t.Errorf("expected 0, got %d", result.IntVal)
	}
}

// --- Nested stripMarks on list with SyntaxObject children ---

func TestStripNodeMarksNestedSyntaxObject(t *testing.T) {
	inner := e.ScopedIdentNode("bar", map[uint64]bool{7: true})
	syntaxObj := &e.Node{Kind: e.SyntaxObjectNode, Quoted: inner}
	list := e.NewListNode([]*e.Node{syntaxObj})
	result := stripMarks(list)
	if result.Kind != e.ListNode {
		t.Fatalf("expected ListNode, got %d", result.Kind)
	}
	child := result.Children[0]
	if child.Kind != e.IdentifierNode || child.Name != "bar" {
		t.Errorf("expected plain identifier 'bar', got %s", child.Repr())
	}
	if len(child.Marks) > 0 {
		t.Error("marks should be stripped from unwrapped SyntaxObject")
	}
}
