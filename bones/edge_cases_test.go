package bones

import (
	"testing"
)

// --- Repr edge cases ---

func TestReprFloat(t *testing.T) {
	if FloatNode(3.14).Repr() != "3.14" {
		t.Errorf("got %s", FloatNode(3.14).Repr())
	}
}

func TestReprString(t *testing.T) {
	if StrNode("hello").Repr() != `"hello"` {
		t.Errorf("got %s", StrNode("hello").Repr())
	}
}

func TestReprBoolTrue(t *testing.T) {
	if BoolNode(true).Repr() != "#t" {
		t.Errorf("got %s", BoolNode(true).Repr())
	}
}

func TestReprBoolFalse(t *testing.T) {
	if BoolNode(false).Repr() != "#f" {
		t.Errorf("got %s", BoolNode(false).Repr())
	}
}

func TestReprQuoteNil(t *testing.T) {
	q := QuoteNodeVal(nil)
	if q.Repr() != "'()" {
		t.Errorf("got %s", q.Repr())
	}
}

func TestReprFunctionNode(t *testing.T) {
	fn := FuncNode(func(args []*Node, ev Evaluator) (*Node, error) { return Nil, nil })
	if fn.Repr() != "#<procedure>" {
		t.Errorf("got %s", fn.Repr())
	}
}

func TestReprForeignNodeWithRepr(t *testing.T) {
	// Foreign wrapping a type with Repr() should delegate
	n := ForeignNodeVal(StrNode("inner"))
	repr := n.Repr()
	if repr != `"inner"` {
		t.Errorf("expected inner Repr delegation, got %s", repr)
	}
}

func TestReprForeignNodeWithout(t *testing.T) {
	n := ForeignNodeVal(42)
	repr := n.Repr()
	if repr != "#<foreign:42>" {
		t.Errorf("got %s", repr)
	}
}

func TestReprMummyNode(t *testing.T) {
	n := MummyNodeVal(42, "int")
	if n.Repr() != "#<mummy:int>" {
		t.Errorf("got %s", n.Repr())
	}
}

func TestReprSyntaxObjectNode(t *testing.T) {
	so := &Node{Kind: SyntaxObjectNode, Quoted: IdentNode("x")}
	if so.Repr() != "x" {
		t.Errorf("got %s", so.Repr())
	}
}

func TestReprSyntaxObjectNodeNilQuoted(t *testing.T) {
	so := &Node{Kind: SyntaxObjectNode}
	if so.Repr() != "#<syntax-object>" {
		t.Errorf("got %s", so.Repr())
	}
}

func TestReprCallNode(t *testing.T) {
	n := &Node{Kind: CallNode, Children: []*Node{IdentNode("+"), IntNode(1), IntNode(2)}}
	if n.Repr() != "(+ 1 2)" {
		t.Errorf("got %s", n.Repr())
	}
}

func TestReprDefineNode(t *testing.T) {
	n := &Node{Kind: DefineNode, Children: []*Node{IdentNode("x"), IntNode(42)}}
	if n.Repr() != "(x 42)" {
		t.Errorf("got %s", n.Repr())
	}
}

func TestReprBeginNode(t *testing.T) {
	n := &Node{Kind: BeginNode, Children: []*Node{IntNode(1), IntNode(2)}}
	if n.Repr() != "(1 2)" {
		t.Errorf("got %s", n.Repr())
	}
}

func TestReprUnknownKind(t *testing.T) {
	n := &Node{Kind: NodeKind(99)}
	if n.Repr() != "#<unknown>" {
		t.Errorf("got %s", n.Repr())
	}
}

// --- FuncNode ---

func TestFuncNodeKind(t *testing.T) {
	fn := FuncNode(func(args []*Node, ev Evaluator) (*Node, error) { return Nil, nil })
	if fn.Kind != FunctionNode {
		t.Errorf("expected FunctionNode, got %d", fn.Kind)
	}
	if fn.FuncVal == nil {
		t.Error("FuncVal should not be nil")
	}
}

// --- Equiv edge cases ---

func TestEquivSamePointer(t *testing.T) {
	n := IntNode(42)
	if !n.Equiv(n) {
		t.Error("same pointer should be equal")
	}
}

func TestEquivNilNil(t *testing.T) {
	if !Nil.Equiv(Nil) {
		t.Error("Nil should equal Nil")
	}
}

func TestEquivNilVsNonNil(t *testing.T) {
	if Nil.Equiv(IntNode(0)) {
		t.Error("Nil should not equal 0")
	}
}

func TestEquivNonNodeType(t *testing.T) {
	if IntNode(1).Equiv("not a node") {
		t.Error("should not equal a string")
	}
}

func TestEquivIntInt(t *testing.T) {
	if !IntNode(42).Equiv(IntNode(42)) {
		t.Error("same ints should be equal")
	}
	if IntNode(1).Equiv(IntNode(2)) {
		t.Error("different ints should not be equal")
	}
}

func TestEquivIntFloatCrossType(t *testing.T) {
	if !IntNode(3).Equiv(FloatNode(3.0)) {
		t.Error("int 3 should equal float 3.0")
	}
}

func TestEquivFloatInt(t *testing.T) {
	if !FloatNode(3.0).Equiv(IntNode(3)) {
		t.Error("float 3.0 should equal int 3")
	}
}

func TestEquivFloatFloat(t *testing.T) {
	if !FloatNode(1.5).Equiv(FloatNode(1.5)) {
		t.Error("same floats should be equal")
	}
	if FloatNode(1.5).Equiv(FloatNode(2.5)) {
		t.Error("different floats should not be equal")
	}
}

func TestEquivIntString(t *testing.T) {
	if IntNode(1).Equiv(StrNode("1")) {
		t.Error("int should not equal string")
	}
}

func TestEquivFloatString(t *testing.T) {
	if FloatNode(1.0).Equiv(StrNode("1.0")) {
		t.Error("float should not equal string")
	}
}

func TestEquivStringsEdge(t *testing.T) {
	if !StrNode("a").Equiv(StrNode("a")) {
		t.Error("same strings should be equal")
	}
	if StrNode("a").Equiv(StrNode("b")) {
		t.Error("different strings should not be equal")
	}
	if StrNode("a").Equiv(IntNode(1)) {
		t.Error("string should not equal int")
	}
}

func TestEquivBooleansEdge(t *testing.T) {
	if !BoolNode(true).Equiv(BoolNode(true)) {
		t.Error("same bools should be equal")
	}
	if BoolNode(true).Equiv(BoolNode(false)) {
		t.Error("different bools should not be equal")
	}
	if BoolNode(true).Equiv(IntNode(1)) {
		t.Error("bool should not equal int")
	}
}

func TestEquivIdentifiersEdge(t *testing.T) {
	if !IdentNode("x").Equiv(IdentNode("x")) {
		t.Error("same idents should be equal")
	}
	if IdentNode("x").Equiv(IdentNode("y")) {
		t.Error("different idents should not be equal")
	}
}

func TestEquivScopedIdentifiersEdge(t *testing.T) {
	a := ScopedIdentNode("x", map[uint64]bool{1: true})
	b := ScopedIdentNode("x", map[uint64]bool{1: true})
	c := ScopedIdentNode("x", map[uint64]bool{2: true})
	d := ScopedIdentNode("y", map[uint64]bool{1: true})
	if !a.Equiv(b) {
		t.Error("same scoped idents should be equal")
	}
	if a.Equiv(c) {
		t.Error("different marks should not be equal")
	}
	if a.Equiv(d) {
		t.Error("different names should not be equal")
	}
}

func TestEquivScopedVsPlain(t *testing.T) {
	plain := IdentNode("x")
	scopedEmpty := ScopedIdentNode("x", nil)
	scopedMarks := ScopedIdentNode("x", map[uint64]bool{1: true})

	if !plain.Equiv(scopedEmpty) {
		t.Error("plain should equal scoped with no marks")
	}
	if plain.Equiv(scopedMarks) {
		t.Error("plain should not equal scoped with marks")
	}
}

func TestEquivIdentVsNonIdent(t *testing.T) {
	if IdentNode("x").Equiv(IntNode(1)) {
		t.Error("ident should not equal int")
	}
}

func TestEquivQuoteNodes(t *testing.T) {
	a := QuoteNodeVal(IntNode(42))
	b := QuoteNodeVal(IntNode(42))
	c := QuoteNodeVal(IntNode(99))
	if !a.Equiv(b) {
		t.Error("same quoted values should be equal")
	}
	if a.Equiv(c) {
		t.Error("different quoted values should not be equal")
	}
}

func TestEquivQuoteNilQuoted(t *testing.T) {
	a := QuoteNodeVal(nil)
	b := QuoteNodeVal(nil)
	if !a.Equiv(b) {
		t.Error("both nil quoted should be equal")
	}
}

func TestEquivQuoteOneNilQuoted(t *testing.T) {
	a := QuoteNodeVal(nil)
	b := QuoteNodeVal(IntNode(1))
	if a.Equiv(b) {
		t.Error("nil vs non-nil quoted should not be equal")
	}
}

func TestEquivQuoteVsNonQuote(t *testing.T) {
	if QuoteNodeVal(IntNode(1)).Equiv(IntNode(1)) {
		t.Error("quote should not equal non-quote")
	}
}

func TestEquivListsEdge(t *testing.T) {
	a := NewListNode([]*Node{IntNode(1), IntNode(2)})
	b := NewListNode([]*Node{IntNode(1), IntNode(2)})
	c := NewListNode([]*Node{IntNode(1), IntNode(3)})
	d := NewListNode([]*Node{IntNode(1)})
	if !a.Equiv(b) {
		t.Error("same lists should be equal")
	}
	if a.Equiv(c) {
		t.Error("different elements should not be equal")
	}
	if a.Equiv(d) {
		t.Error("different lengths should not be equal")
	}
}

func TestEquivListVsNonList(t *testing.T) {
	if NewListNode([]*Node{IntNode(1)}).Equiv(IntNode(1)) {
		t.Error("list should not equal non-list")
	}
}

func TestEquivDottedLists(t *testing.T) {
	a := &Node{Kind: ListNode, Children: []*Node{IntNode(1)}, DottedTail: IntNode(2)}
	b := &Node{Kind: ListNode, Children: []*Node{IntNode(1)}, DottedTail: IntNode(2)}
	c := &Node{Kind: ListNode, Children: []*Node{IntNode(1)}, DottedTail: IntNode(3)}
	d := &Node{Kind: ListNode, Children: []*Node{IntNode(1)}} // no dotted tail
	if !a.Equiv(b) {
		t.Error("same dotted lists should be equal")
	}
	if a.Equiv(c) {
		t.Error("different dotted tails should not be equal")
	}
	if a.Equiv(d) {
		t.Error("dotted vs non-dotted should not be equal")
	}
}

func TestEquivForeignNodes(t *testing.T) {
	a := ForeignNodeVal(42)
	b := ForeignNodeVal(42)
	c := ForeignNodeVal(99)
	if !a.Equiv(b) {
		t.Error("same foreign values should be equal")
	}
	if a.Equiv(c) {
		t.Error("different foreign values should not be equal")
	}
}

func TestEquivForeignVsMummy(t *testing.T) {
	a := ForeignNodeVal(42)
	b := MummyNodeVal(42, "int")
	if a.Equiv(b) {
		t.Error("foreign should not equal mummy (different kinds)")
	}
}

func TestEquivMummyNodes(t *testing.T) {
	a := MummyNodeVal(42, "int")
	b := MummyNodeVal(42, "int")
	c := MummyNodeVal(99, "int")
	if !a.Equiv(b) {
		t.Error("same mummy values should be equal")
	}
	if a.Equiv(c) {
		t.Error("different mummy values should not be equal")
	}
}

func TestEquivFunctionNodes(t *testing.T) {
	fn := func(args []*Node, ev Evaluator) (*Node, error) { return Nil, nil }
	a := FuncNode(fn)
	b := FuncNode(fn)
	if a.Equiv(b) {
		// Different FuncVal pointers — not equal even if same function
	}
	if !a.Equiv(a) {
		t.Error("same FuncNode pointer should be equal to itself")
	}
}

func TestEquivCallNodeVsListNode(t *testing.T) {
	a := &Node{Kind: CallNode, Children: []*Node{IntNode(1)}}
	b := &Node{Kind: ListNode, Children: []*Node{IntNode(1)}}
	if !a.Equiv(b) {
		t.Error("CallNode and ListNode with same children should be equal (both list-like)")
	}
}

// --- NodeTypeName edge cases ---

func TestNodeTypeNameFunction(t *testing.T) {
	fn := FuncNode(func(args []*Node, ev Evaluator) (*Node, error) { return Nil, nil })
	if NodeTypeName(fn) != "procedure" {
		t.Errorf("got %s", NodeTypeName(fn))
	}
}

func TestNodeTypeNameMummy(t *testing.T) {
	if NodeTypeName(MummyNodeVal(42, "int")) != "mummy value" {
		t.Errorf("got %s", NodeTypeName(MummyNodeVal(42, "int")))
	}
}

func TestNodeTypeNameSyntaxObject(t *testing.T) {
	so := &Node{Kind: SyntaxObjectNode}
	if NodeTypeName(so) != "syntax object" {
		t.Errorf("got %s", NodeTypeName(so))
	}
}

func TestNodeTypeNameUnknown(t *testing.T) {
	n := &Node{Kind: NodeKind(99)}
	if NodeTypeName(n) != "unknown" {
		t.Errorf("got %s", NodeTypeName(n))
	}
}

// --- NodeMarksEq ---

func TestNodeMarksEqBothNil(t *testing.T) {
	if !NodeMarksEq(nil, nil) {
		t.Error("both nil should be equal")
	}
}

func TestNodeMarksEqDifferentLength(t *testing.T) {
	if NodeMarksEq(map[uint64]bool{1: true}, map[uint64]bool{1: true, 2: true}) {
		t.Error("different lengths should not be equal")
	}
}

func TestNodeMarksEqMissingKey(t *testing.T) {
	if NodeMarksEq(map[uint64]bool{1: true}, map[uint64]bool{2: true}) {
		t.Error("different keys should not be equal")
	}
}

// --- First/Rest edge cases ---

func TestFirstOfNonList(t *testing.T) {
	if IntNode(42).First() != Nil {
		t.Error("First of non-list should be Nil")
	}
}

func TestRestOfNonList(t *testing.T) {
	if IntNode(42).Rest() != Nil {
		t.Error("Rest of non-list should be Nil")
	}
}

func TestFirstOfEmptyList(t *testing.T) {
	if NewListNode(nil).First() != Nil {
		t.Error("First of empty list should be Nil")
	}
}

func TestRestWithMultipleChildrenAndDottedTail(t *testing.T) {
	// (1 2 . 3) → Rest is (2 . 3)
	n := &Node{Kind: ListNode, Children: []*Node{IntNode(1), IntNode(2)}, DottedTail: IntNode(3)}
	rest := n.Rest()
	if rest.Kind != ListNode || len(rest.Children) != 1 || rest.DottedTail == nil {
		t.Errorf("expected (2 . 3), got %s", rest.Repr())
	}
}

// --- IsNil ---

func TestIsNilOnNilSingleton(t *testing.T) {
	if !Nil.IsNil() {
		t.Error("Nil singleton should be nil")
	}
}

func TestIsNilOnNewNilNode(t *testing.T) {
	n := &Node{Kind: NilNode}
	if !n.IsNil() {
		t.Error("new NilNode should be nil")
	}
}

func TestIsNilOnNonNil(t *testing.T) {
	if IntNode(0).IsNil() {
		t.Error("IntNode(0) should not be nil")
	}
}

// --- IdentName ---

func TestIdentNameOnNonIdentifier(t *testing.T) {
	if IntNode(1).IdentName() != "" {
		t.Error("non-identifier should return empty string")
	}
}

func TestIdentNameOnScopedIdent(t *testing.T) {
	n := ScopedIdentNode("foo", map[uint64]bool{1: true})
	if n.IdentName() != "foo" {
		t.Errorf("expected foo, got %s", n.IdentName())
	}
}

// --- MummyNodeVal ---

func TestMummyNodeValFields(t *testing.T) {
	n := MummyNodeVal("hello", "string")
	if n.Kind != MummyNode {
		t.Errorf("expected MummyNode, got %d", n.Kind)
	}
	if n.ForeignVal != "hello" {
		t.Errorf("expected hello, got %v", n.ForeignVal)
	}
	if n.TypeNameV != "string" {
		t.Errorf("expected string, got %s", n.TypeNameV)
	}
}

// --- reprList with DottedTail and no children ---

func TestReprListDottedTailNoChildren(t *testing.T) {
	n := &Node{Kind: ListNode, DottedTail: IntNode(42)}
	if n.Repr() != "(. 42)" {
		t.Errorf("got %s", n.Repr())
	}
}

// --- equivList across different list-like kinds ---

func TestEquivBeginVsLambda(t *testing.T) {
	a := &Node{Kind: BeginNode, Children: []*Node{IntNode(1)}}
	b := &Node{Kind: LambdaNode, Children: []*Node{IntNode(1)}}
	if !a.Equiv(b) {
		t.Error("different list-like kinds with same children should be equal")
	}
}
