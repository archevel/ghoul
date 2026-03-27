package macromancy

import (
	"testing"

	"github.com/archevel/ghoul/bones"
)

func TestWrapSyntaxLeaf(t *testing.T) {
	n := bones.IdentNode("x")
	marks := MarkSet{1: true}
	wrapped := WrapSyntax(n, marks)
	if wrapped.Kind != bones.SyntaxObjectNode {
		t.Errorf("expected SyntaxObjectNode, got %d", wrapped.Kind)
	}
	if wrapped.Quoted.Name != "x" {
		t.Errorf("expected datum x, got %s", wrapped.Quoted.Name)
	}
	if !wrapped.Marks[1] {
		t.Error("expected mark 1")
	}
}

func TestWrapSyntaxList(t *testing.T) {
	n := bones.NewListNode([]*bones.Node{
		bones.IdentNode("x"),
		bones.IntNode(42),
	})
	wrapped := WrapSyntax(n, MarkSet{1: true})
	if wrapped.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", wrapped.Kind)
	}
	if len(wrapped.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(wrapped.Children))
	}
	if wrapped.Children[0].Kind != bones.SyntaxObjectNode {
		t.Error("expected child 0 to be SyntaxObjectNode")
	}
	if wrapped.Children[1].Kind != bones.SyntaxObjectNode {
		t.Error("expected child 1 to be SyntaxObjectNode")
	}
}

func TestWrapSyntaxNil(t *testing.T) {
	wrapped := WrapSyntax(bones.Nil, MarkSet{1: true})
	if !wrapped.IsNil() {
		t.Error("expected Nil")
	}
}

func TestApplyMarkOnSyntaxObject(t *testing.T) {
	so := &bones.Node{
		Kind:   bones.SyntaxObjectNode,
		Quoted: bones.IdentNode("x"),
		Marks:  map[uint64]bool{},
	}
	marked := ApplyMark(so, 1)
	if !marked.Marks[1] {
		t.Error("expected mark 1 toggled on")
	}

	// Toggle again → mark 1 should be gone
	unmarked := ApplyMark(marked, 1)
	if unmarked.Marks[1] {
		t.Error("expected mark 1 toggled off")
	}
}

func TestApplyMarkOnPlainIdent(t *testing.T) {
	n := bones.IdentNode("x")
	marked := ApplyMark(n, 1)
	if marked.Kind != bones.IdentifierNode {
		t.Errorf("expected IdentifierNode, got %d", marked.Kind)
	}
	if !marked.Marks[1] {
		t.Error("expected mark 1")
	}
}

func TestApplyMarkOnList(t *testing.T) {
	n := bones.NewListNode([]*bones.Node{
		bones.IdentNode("x"),
		bones.IntNode(42),
	})
	marked := ApplyMark(n, 1)
	if marked.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", marked.Kind)
	}
	// x should be marked
	if !marked.Children[0].Marks[1] {
		t.Error("expected mark on identifier child")
	}
	// 42 should be unchanged
	if marked.Children[1].Kind != bones.IntegerNode {
		t.Error("expected IntegerNode unchanged")
	}
}

func TestResolveSyntaxSyntaxObjectWithMarks(t *testing.T) {
	so := &bones.Node{
		Kind:   bones.SyntaxObjectNode,
		Quoted: bones.IdentNode("x"),
		Marks:  map[uint64]bool{1: true},
	}
	resolved := ResolveSyntax(so)
	if resolved.Kind != bones.IdentifierNode {
		t.Fatalf("expected IdentifierNode, got %d", resolved.Kind)
	}
	if resolved.Name != "x" {
		t.Errorf("expected x, got %s", resolved.Name)
	}
	if !resolved.Marks[1] {
		t.Error("expected mark 1 on resolved identifier")
	}
}

func TestResolveSyntaxSyntaxObjectNoMarks(t *testing.T) {
	so := &bones.Node{
		Kind:   bones.SyntaxObjectNode,
		Quoted: bones.IdentNode("x"),
		Marks:  map[uint64]bool{},
	}
	resolved := ResolveSyntax(so)
	if resolved.Kind != bones.IdentifierNode {
		t.Fatalf("expected IdentifierNode, got %d", resolved.Kind)
	}
	if len(resolved.Marks) > 0 {
		t.Error("expected no marks on resolved identifier")
	}
}

func TestResolveSyntaxList(t *testing.T) {
	n := bones.NewListNode([]*bones.Node{
		&bones.Node{
			Kind:   bones.SyntaxObjectNode,
			Quoted: bones.IdentNode("x"),
			Marks:  map[uint64]bool{1: true},
		},
		bones.IntNode(42),
	})
	resolved := ResolveSyntax(n)
	if resolved.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", resolved.Kind)
	}
	if resolved.Children[0].Kind != bones.IdentifierNode {
		t.Error("expected resolved identifier")
	}
	if resolved.Children[1].Kind != bones.IntegerNode {
		t.Error("expected integer unchanged")
	}
}

func TestWrapApplyResolveRoundTrip(t *testing.T) {
	// Simulate the general transformer lifecycle:
	// 1. Wrap input with marks
	// 2. Apply a mark (toggle)
	// 3. Resolve
	input := bones.NewListNode([]*bones.Node{
		bones.IdentNode("my-macro"),
		bones.IdentNode("x"),
		bones.IntNode(42),
	})

	// Wrap: all leaves get SyntaxObject with empty marks
	wrapped := WrapSyntax(input, NewMarkSet())

	// Apply mark 1: SyntaxObjects with identifiers get mark 1 toggled
	marked := ApplyMark(wrapped, 1)

	// Apply mark 1 again (output mark): toggles off for input identifiers
	doubleMarked := ApplyMark(marked, 1)

	// Resolve: strip SyntaxObjects
	resolved := ResolveSyntax(doubleMarked)

	if resolved.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", resolved.Kind)
	}
	// Identifiers should have no marks (toggled on then off)
	for _, child := range resolved.Children {
		if child.Kind == bones.IdentifierNode && len(child.Marks) > 0 {
			t.Errorf("expected no marks on %s after double toggle, got %v", child.Name, child.Marks)
		}
	}
}
