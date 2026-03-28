package bones

import (
	"fmt"
	"strconv"
	"strings"
)

type NodeKind int

const (
	// Syntax nodes (parser output)
	NilNode NodeKind = iota
	IntegerNode
	FloatNodeKind
	StringNode
	BooleanNode
	IdentifierNode
	QuoteNode
	ListNode

	// Semantic nodes (reanimator output)
	DefineNode
	SetNode
	LambdaNode
	CondNode
	BeginNode
	CallNode

	// Runtime nodes
	FunctionNode
	ForeignNode
	MummyNode
	SyntaxObjectNode
)

// Evaluator is a forward declaration to break the import cycle.
// The consume package sets this to *consume.Evaluator at init time.
type Evaluator = interface{}

type Node struct {
	Kind NodeKind
	Loc  CodeLocation

	// Value storage (one used per kind)
	IntVal   int64
	FloatVal float64
	StrVal   string
	BoolVal  bool

	// Identifier
	Name  string
	Marks map[uint64]bool // non-nil = scoped identifier

	// List structure
	Children   []*Node
	DottedTail *Node // non-nil for improper lists

	// Quote / SyntaxObject datum
	Quoted *Node

	// Lambda
	Params *ParamSpec

	// Cond
	Clauses []*CondClause

	// Function closure
	FuncVal *func([]*Node, Evaluator) (*Node, error)

	// Foreign/Mummy values
	ForeignVal any
	TypeNameV  string // MummyNode type name
}

type ParamSpec struct {
	Fixed    []*Node
	Variadic *Node // nil if no rest param
}

type CondClause struct {
	Test       *Node
	Consequent []*Node
	IsElse     bool
}

// Nil is the singleton empty list / void value.
var Nil = &Node{Kind: NilNode}

// --- Constructors ---

// Pre-allocated singletons for common values to reduce allocation pressure.
var (
	trueNode  = &Node{Kind: BooleanNode, BoolVal: true}
	falseNode = &Node{Kind: BooleanNode, BoolVal: false}
)

// Small integer cache for -128..127 (same range as Java's Integer cache).
const (
	intCacheMin = -128
	intCacheMax = 127
)

var intCache [intCacheMax - intCacheMin + 1]*Node

func init() {
	for i := int64(intCacheMin); i <= intCacheMax; i++ {
		intCache[i-intCacheMin] = &Node{Kind: IntegerNode, IntVal: i}
	}
}

func IntNode(v int64) *Node {
	if v >= intCacheMin && v <= intCacheMax {
		return intCache[v-intCacheMin]
	}
	return &Node{Kind: IntegerNode, IntVal: v}
}

func FloatNode(v float64) *Node {
	return &Node{Kind: FloatNodeKind, FloatVal: v}
}

func StrNode(v string) *Node {
	return &Node{Kind: StringNode, StrVal: v}
}

func BoolNode(v bool) *Node {
	if v {
		return trueNode
	}
	return falseNode
}

func IdentNode(name string) *Node {
	return &Node{Kind: IdentifierNode, Name: name}
}

func ScopedIdentNode(name string, marks map[uint64]bool) *Node {
	return &Node{Kind: IdentifierNode, Name: name, Marks: marks}
}

func NewListNode(children []*Node) *Node {
	return &Node{Kind: ListNode, Children: children}
}

func QuoteNodeVal(datum *Node) *Node {
	return &Node{Kind: QuoteNode, Quoted: datum}
}

func ForeignNodeVal(val any) *Node {
	return &Node{Kind: ForeignNode, ForeignVal: val}
}

func MummyNodeVal(val any, typeName string) *Node {
	return &Node{Kind: MummyNode, ForeignVal: val, TypeNameV: typeName}
}

func FuncNode(fn func([]*Node, Evaluator) (*Node, error)) *Node {
	return &Node{Kind: FunctionNode, FuncVal: &fn}
}

// --- Accessors ---

func (n *Node) IsNil() bool {
	return n == Nil || n.Kind == NilNode
}

func (n *Node) IdentName() string {
	if n.Kind == IdentifierNode {
		return n.Name
	}
	return ""
}

// First returns the first child of a list node, or Nil.
func (n *Node) First() *Node {
	if n.Kind == ListNode && len(n.Children) > 0 {
		return n.Children[0]
	}
	return Nil
}

// Rest returns a new list node with Children[1:], handling dotted tails.
func (n *Node) Rest() *Node {
	if n.Kind != ListNode || len(n.Children) == 0 {
		return Nil
	}
	if len(n.Children) == 1 {
		if n.DottedTail != nil {
			return n.DottedTail
		}
		return Nil
	}
	return &Node{
		Kind:       ListNode,
		Children:   n.Children[1:],
		DottedTail: n.DottedTail,
	}
}

// --- Repr ---

func (n *Node) Repr() string {
	switch n.Kind {
	case NilNode:
		return "()"
	case IntegerNode:
		return strconv.FormatInt(n.IntVal, 10)
	case FloatNodeKind:
		return strconv.FormatFloat(n.FloatVal, 'g', -1, 64)
	case StringNode:
		return `"` + n.StrVal + `"`
	case BooleanNode:
		if n.BoolVal {
			return "#t"
		}
		return "#f"
	case IdentifierNode:
		return n.Name
	case QuoteNode:
		if n.Quoted != nil {
			return "'" + n.Quoted.Repr()
		}
		return "'()"
	case ListNode, CallNode, DefineNode, SetNode, LambdaNode,
		CondNode, BeginNode:
		return reprList(n)
	case FunctionNode:
		return "#<procedure>"
	case ForeignNode:
		type reprable interface{ Repr() string }
		if r, ok := n.ForeignVal.(reprable); ok {
			return r.Repr()
		}
		return fmt.Sprintf("#<foreign:%#v>", n.ForeignVal)
	case MummyNode:
		return "#<mummy:" + n.TypeNameV + ">"
	case SyntaxObjectNode:
		if n.Quoted != nil {
			return n.Quoted.Repr()
		}
		return "#<syntax-object>"
	default:
		return "#<unknown>"
	}
}

func reprList(n *Node) string {
	var b strings.Builder
	b.WriteRune('(')
	for i, child := range n.Children {
		if i > 0 {
			b.WriteRune(' ')
		}
		b.WriteString(child.Repr())
	}
	if n.DottedTail != nil {
		if len(n.Children) > 0 {
			b.WriteRune(' ')
		}
		b.WriteString(". ")
		b.WriteString(n.DottedTail.Repr())
	}
	b.WriteRune(')')
	return b.String()
}

// --- Equiv ---

// Equiv compares this node to another value. Accepts *Node.
func (n *Node) Equiv(other any) bool {
	if otherNode, ok := other.(*Node); ok {
		return n.equivNode(otherNode)
	}
	return false
}

func (n *Node) equivNode(other *Node) bool {
	if n == other {
		return true
	}
	if n.Kind == NilNode && other.Kind == NilNode {
		return true
	}

	switch n.Kind {
	case IntegerNode:
		switch other.Kind {
		case IntegerNode:
			return n.IntVal == other.IntVal
		case FloatNodeKind:
			return float64(n.IntVal) == other.FloatVal
		}
	case FloatNodeKind:
		switch other.Kind {
		case FloatNodeKind:
			return n.FloatVal == other.FloatVal
		case IntegerNode:
			return n.FloatVal == float64(other.IntVal)
		}
	case StringNode:
		return other.Kind == StringNode && n.StrVal == other.StrVal
	case BooleanNode:
		return other.Kind == BooleanNode && n.BoolVal == other.BoolVal
	case IdentifierNode:
		if other.Kind != IdentifierNode {
			return false
		}
		if n.Name != other.Name {
			return false
		}
		// Both plain or both have matching marks
		nScoped := len(n.Marks) > 0
		oScoped := len(other.Marks) > 0
		if !nScoped && !oScoped {
			return true
		}
		if nScoped && oScoped {
			return NodeMarksEq(n.Marks, other.Marks)
		}
		// One scoped, one plain: equal only if marks are empty
		if nScoped {
			return len(n.Marks) == 0
		}
		return len(other.Marks) == 0
	case QuoteNode:
		if other.Kind != QuoteNode {
			return false
		}
		if n.Quoted == nil && other.Quoted == nil {
			return true
		}
		if n.Quoted == nil || other.Quoted == nil {
			return false
		}
		return n.Quoted.Equiv(other.Quoted)
	case ListNode, CallNode, DefineNode, SetNode, LambdaNode,
		CondNode, BeginNode:
		return equivList(n, other)
	case ForeignNode, MummyNode:
		if other.Kind != n.Kind {
			return false
		}
		return n.ForeignVal == other.ForeignVal
	case FunctionNode:
		return other.Kind == FunctionNode && n.FuncVal == other.FuncVal
	}
	return false
}

func equivList(a, b *Node) bool {
	if !isListLike(b) {
		return false
	}
	if len(a.Children) != len(b.Children) {
		return false
	}
	for i := range a.Children {
		if !a.Children[i].Equiv(b.Children[i]) {
			return false
		}
	}
	if a.DottedTail == nil && b.DottedTail == nil {
		return true
	}
	if a.DottedTail == nil || b.DottedTail == nil {
		return false
	}
	return a.DottedTail.Equiv(b.DottedTail)
}

func isListLike(n *Node) bool {
	switch n.Kind {
	case ListNode, CallNode, DefineNode, SetNode, LambdaNode,
		CondNode, BeginNode:
		return true
	}
	return false
}


// --- Helpers ---

func NodeMarksEq(a, b map[uint64]bool) bool {
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

func NodeTypeName(n *Node) string {
	switch n.Kind {
	case NilNode:
		return "empty list"
	case IntegerNode:
		return "integer"
	case FloatNodeKind:
		return "float"
	case StringNode:
		return "string"
	case BooleanNode:
		return "boolean"
	case IdentifierNode:
		return "identifier"
	case QuoteNode:
		return "quoted expression"
	case ListNode, CallNode, DefineNode, SetNode, LambdaNode,
		CondNode, BeginNode:
		return "list"
	case FunctionNode:
		return "procedure"
	case ForeignNode:
		return "foreign value"
	case MummyNode:
		return "mummy value"
	case SyntaxObjectNode:
		return "syntax object"
	default:
		return "unknown"
	}
}
