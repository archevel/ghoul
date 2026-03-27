package bones

import (
	"testing"
)

func TestIntNode(t *testing.T) {
	n := IntNode(42)
	if n.Kind != IntegerNode {
		t.Errorf("expected IntegerNode, got %d", n.Kind)
	}
	if n.IntVal != 42 {
		t.Errorf("expected 42, got %d", n.IntVal)
	}
	if n.Repr() != "42" {
		t.Errorf("expected '42', got '%s'", n.Repr())
	}
}

func TestFloatNode(t *testing.T) {
	n := FloatNode(3.14)
	if n.Kind != FloatNodeKind {
		t.Errorf("expected FloatNodeKind, got %d", n.Kind)
	}
	if n.Repr() != "3.14" {
		t.Errorf("expected '3.14', got '%s'", n.Repr())
	}
}

func TestStrNode(t *testing.T) {
	n := StrNode("hello")
	if n.Kind != StringNode {
		t.Errorf("expected StringNode, got %d", n.Kind)
	}
	if n.Repr() != `"hello"` {
		t.Errorf(`expected '"hello"', got '%s'`, n.Repr())
	}
}

func TestBoolNode(t *testing.T) {
	tr := BoolNode(true)
	fa := BoolNode(false)
	if tr.Repr() != "#t" {
		t.Errorf("expected #t, got %s", tr.Repr())
	}
	if fa.Repr() != "#f" {
		t.Errorf("expected #f, got %s", fa.Repr())
	}
}

func TestIdentNode(t *testing.T) {
	n := IdentNode("foo")
	if n.Kind != IdentifierNode {
		t.Errorf("expected IdentifierNode, got %d", n.Kind)
	}
	if n.Name != "foo" {
		t.Errorf("expected 'foo', got '%s'", n.Name)
	}
	if n.Repr() != "foo" {
		t.Errorf("expected 'foo', got '%s'", n.Repr())
	}
}

func TestScopedIdentNode(t *testing.T) {
	n := ScopedIdentNode("x", map[uint64]bool{1: true, 2: true})
	if n.Kind != IdentifierNode {
		t.Errorf("expected IdentifierNode, got %d", n.Kind)
	}
	if n.Marks == nil || len(n.Marks) != 2 {
		t.Errorf("expected 2 marks, got %v", n.Marks)
	}
	if n.Repr() != "x" {
		t.Errorf("expected 'x', got '%s'", n.Repr())
	}
}

func TestNilNode(t *testing.T) {
	n := Nil
	if n.Kind != NilNode {
		t.Errorf("expected NilNode, got %d", n.Kind)
	}
	if n.Repr() != "()" {
		t.Errorf("expected '()', got '%s'", n.Repr())
	}
	if !n.IsNil() {
		t.Error("expected IsNil() to be true")
	}
}

func TestListNodeRepr(t *testing.T) {
	// (1 2 3)
	n := NewListNode([]*Node{IntNode(1), IntNode(2), IntNode(3)})
	if n.Repr() != "(1 2 3)" {
		t.Errorf("expected '(1 2 3)', got '%s'", n.Repr())
	}
}

func TestNestedListRepr(t *testing.T) {
	// (+ (- 3 1) 2)
	inner := NewListNode([]*Node{IdentNode("-"), IntNode(3), IntNode(1)})
	outer := NewListNode([]*Node{IdentNode("+"), inner, IntNode(2)})
	if outer.Repr() != "(+ (- 3 1) 2)" {
		t.Errorf("expected '(+ (- 3 1) 2)', got '%s'", outer.Repr())
	}
}

func TestEmptyListRepr(t *testing.T) {
	n := NewListNode(nil)
	if n.Repr() != "()" {
		t.Errorf("expected '()', got '%s'", n.Repr())
	}
}

func TestDottedPairRepr(t *testing.T) {
	// (1 . 2)
	n := &Node{Kind: ListNode, Children: []*Node{IntNode(1)}, DottedTail: IntNode(2)}
	if n.Repr() != "(1 . 2)" {
		t.Errorf("expected '(1 . 2)', got '%s'", n.Repr())
	}
}

func TestDottedListRepr(t *testing.T) {
	// (1 2 . 3)
	n := &Node{Kind: ListNode, Children: []*Node{IntNode(1), IntNode(2)}, DottedTail: IntNode(3)}
	if n.Repr() != "(1 2 . 3)" {
		t.Errorf("expected '(1 2 . 3)', got '%s'", n.Repr())
	}
}

func TestQuoteNodeRepr(t *testing.T) {
	n := QuoteNodeVal(IntNode(42))
	if n.Repr() != "'42" {
		t.Errorf("expected \"'42\", got '%s'", n.Repr())
	}
}

func TestFirstAndRest(t *testing.T) {
	n := NewListNode([]*Node{IntNode(1), IntNode(2), IntNode(3)})

	first := n.First()
	if first.IntVal != 1 {
		t.Errorf("expected First()=1, got %d", first.IntVal)
	}

	rest := n.Rest()
	if rest.Kind != ListNode {
		t.Errorf("expected ListNode, got %d", rest.Kind)
	}
	if len(rest.Children) != 2 {
		t.Fatalf("expected 2 children in Rest(), got %d", len(rest.Children))
	}
	if rest.Children[0].IntVal != 2 {
		t.Errorf("expected 2, got %d", rest.Children[0].IntVal)
	}
}

func TestFirstOfNil(t *testing.T) {
	if Nil.First() != Nil {
		t.Error("First() of Nil should be Nil")
	}
}

func TestRestOfNil(t *testing.T) {
	if Nil.Rest() != Nil {
		t.Error("Rest() of Nil should be Nil")
	}
}

func TestRestOfSingleElement(t *testing.T) {
	n := NewListNode([]*Node{IntNode(1)})
	rest := n.Rest()
	if !rest.IsNil() {
		t.Errorf("expected Nil for rest of single-element list, got %s", rest.Repr())
	}
}

func TestRestOfDottedPair(t *testing.T) {
	// (1 . 2) -> cdr is 2
	n := &Node{Kind: ListNode, Children: []*Node{IntNode(1)}, DottedTail: IntNode(2)}
	rest := n.Rest()
	if rest.Kind != IntegerNode || rest.IntVal != 2 {
		t.Errorf("expected 2, got %s", rest.Repr())
	}
}

func TestRestOfDottedList(t *testing.T) {
	// (1 2 . 3) -> cdr is (2 . 3)
	n := &Node{Kind: ListNode, Children: []*Node{IntNode(1), IntNode(2)}, DottedTail: IntNode(3)}
	rest := n.Rest()
	if rest.Kind != ListNode {
		t.Fatalf("expected ListNode, got %d", rest.Kind)
	}
	if len(rest.Children) != 1 || rest.Children[0].IntVal != 2 {
		t.Errorf("expected (2 . 3), got %s", rest.Repr())
	}
	if rest.DottedTail == nil || rest.DottedTail.IntVal != 3 {
		t.Errorf("expected DottedTail=3, got %v", rest.DottedTail)
	}
}

func TestEquivIntegers(t *testing.T) {
	a := IntNode(42)
	b := IntNode(42)
	c := IntNode(99)
	if !a.Equiv(b) {
		t.Error("42 should equal 42")
	}
	if a.Equiv(c) {
		t.Error("42 should not equal 99")
	}
}

func TestEquivIntFloat(t *testing.T) {
	a := IntNode(3)
	b := FloatNode(3.0)
	if !a.Equiv(b) {
		t.Error("int 3 should equal float 3.0")
	}
}

func TestEquivStrings(t *testing.T) {
	a := StrNode("hello")
	b := StrNode("hello")
	c := StrNode("world")
	if !a.Equiv(b) {
		t.Error("same strings should be equal")
	}
	if a.Equiv(c) {
		t.Error("different strings should not be equal")
	}
}

func TestEquivBooleans(t *testing.T) {
	if !BoolNode(true).Equiv(BoolNode(true)) {
		t.Error("#t should equal #t")
	}
	if BoolNode(true).Equiv(BoolNode(false)) {
		t.Error("#t should not equal #f")
	}
}

func TestEquivIdentifiers(t *testing.T) {
	a := IdentNode("x")
	b := IdentNode("x")
	c := IdentNode("y")
	if !a.Equiv(b) {
		t.Error("same identifiers should be equal")
	}
	if a.Equiv(c) {
		t.Error("different identifiers should not be equal")
	}
}

func TestEquivScopedIdentifiers(t *testing.T) {
	a := ScopedIdentNode("x", map[uint64]bool{1: true})
	b := ScopedIdentNode("x", map[uint64]bool{1: true})
	c := ScopedIdentNode("x", map[uint64]bool{2: true})
	if !a.Equiv(b) {
		t.Error("same scoped identifiers should be equal")
	}
	if a.Equiv(c) {
		t.Error("different marks should not be equal")
	}
}

func TestEquivScopedAndPlain(t *testing.T) {
	plain := IdentNode("x")
	scoped := ScopedIdentNode("x", nil)
	if !plain.Equiv(scoped) {
		t.Error("plain id should equal scoped id with no marks")
	}
}

func TestEquivLists(t *testing.T) {
	a := NewListNode([]*Node{IntNode(1), IntNode(2)})
	b := NewListNode([]*Node{IntNode(1), IntNode(2)})
	c := NewListNode([]*Node{IntNode(1), IntNode(3)})
	if !a.Equiv(b) {
		t.Error("same lists should be equal")
	}
	if a.Equiv(c) {
		t.Error("different lists should not be equal")
	}
}

func TestEquivNil(t *testing.T) {
	if !Nil.Equiv(Nil) {
		t.Error("Nil should equal Nil")
	}
	if Nil.Equiv(IntNode(0)) {
		t.Error("Nil should not equal 0")
	}
}

func TestEquivDifferentKinds(t *testing.T) {
	if IntNode(1).Equiv(StrNode("1")) {
		t.Error("int should not equal string")
	}
}

func TestNodeTypeNameHelper(t *testing.T) {
	tests := []struct {
		node     *Node
		expected string
	}{
		{IntNode(1), "integer"},
		{FloatNode(1.0), "float"},
		{StrNode("a"), "string"},
		{BoolNode(true), "boolean"},
		{IdentNode("x"), "identifier"},
		{ScopedIdentNode("x", map[uint64]bool{1: true}), "identifier"},
		{QuoteNodeVal(IntNode(1)), "quoted expression"},
		{NewListNode([]*Node{IntNode(1)}), "list"},
		{Nil, "empty list"},
		{ForeignNodeVal(42), "foreign value"},
	}
	for _, tt := range tests {
		got := NodeTypeName(tt.node)
		if got != tt.expected {
			t.Errorf("NodeTypeName(%s) = %s, want %s", tt.node.Repr(), got, tt.expected)
		}
	}
}

func TestIsNil(t *testing.T) {
	if !Nil.IsNil() {
		t.Error("Nil should be nil")
	}
	if IntNode(0).IsNil() {
		t.Error("0 should not be nil")
	}
	if NewListNode(nil).IsNil() {
		t.Error("empty ListNode should not be nil (it's a list, not nil)")
	}
}

func TestIdentName(t *testing.T) {
	if IdentNode("foo").IdentName() != "foo" {
		t.Error("expected foo")
	}
	if ScopedIdentNode("bar", map[uint64]bool{1: true}).IdentName() != "bar" {
		t.Error("expected bar")
	}
	if IntNode(1).IdentName() != "" {
		t.Error("non-identifier should return empty string")
	}
}

func TestForeignNodeRepr(t *testing.T) {
	n := ForeignNodeVal("test")
	if n.Kind != ForeignNode {
		t.Errorf("expected ForeignNode, got %d", n.Kind)
	}
}
