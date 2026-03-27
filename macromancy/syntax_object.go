package macromancy

import (
	"github.com/archevel/ghoul/bones"
)

// WrapSyntax wraps leaf nodes as SyntaxObjectNodes while preserving list
// structure, so Children-based traversal continues to work.
func WrapSyntax(node *bones.Node, marks MarkSet) *bones.Node {
	if node == nil || node.IsNil() {
		return bones.Nil
	}
	switch node.Kind {
	case bones.ListNode:
		children := make([]*bones.Node, len(node.Children))
		for i, child := range node.Children {
			children[i] = WrapSyntax(child, marks)
		}
		result := bones.NewListNode(children)
		result.Loc = node.Loc
		if node.DottedTail != nil {
			result.DottedTail = WrapSyntax(node.DottedTail, marks)
		}
		return result
	default:
		return &bones.Node{
			Kind:   bones.SyntaxObjectNode,
			Quoted: node,
			Marks:  copyMarks(marks),
		}
	}
}

// ApplyMark toggles a mark on identifier nodes in the tree.
func ApplyMark(node *bones.Node, mark Mark) *bones.Node {
	if node == nil || node.IsNil() {
		return node
	}
	switch node.Kind {
	case bones.SyntaxObjectNode:
		if node.Quoted != nil && node.Quoted.Kind == bones.IdentifierNode {
			newMarks := MarkSet(node.Marks).Toggle(mark)
			return &bones.Node{
				Kind:   bones.SyntaxObjectNode,
				Quoted: node.Quoted,
				Marks:  newMarks,
			}
		}
		return node
	case bones.IdentifierNode:
		if len(node.Marks) > 0 {
			newMarks := MarkSet(node.Marks).Toggle(mark)
			return bones.ScopedIdentNode(node.Name, newMarks)
		}
		return bones.ScopedIdentNode(node.Name, map[uint64]bool{mark: true})
	case bones.ListNode:
		children := make([]*bones.Node, len(node.Children))
		for i, child := range node.Children {
			children[i] = ApplyMark(child, mark)
		}
		result := &bones.Node{Kind: bones.ListNode, Children: children, Loc: node.Loc}
		if node.DottedTail != nil {
			result.DottedTail = ApplyMark(node.DottedTail, mark)
		}
		return result
	default:
		return node
	}
}

// ResolveSyntax strips SyntaxObjectNode wrappers, converting marked identifiers
// to scoped identifiers and unmarked ones back to plain identifiers.
func ResolveSyntax(node *bones.Node) *bones.Node {
	if node == nil || node.IsNil() {
		return node
	}
	switch node.Kind {
	case bones.SyntaxObjectNode:
		if node.Quoted != nil && node.Quoted.Kind == bones.IdentifierNode {
			marks := MarkSet(node.Marks)
			if marks.IsEmpty() {
				return bones.IdentNode(node.Quoted.Name)
			}
			return bones.ScopedIdentNode(node.Quoted.Name, node.Marks)
		}
		if node.Quoted != nil {
			return node.Quoted
		}
		return bones.Nil
	case bones.ListNode:
		children := make([]*bones.Node, len(node.Children))
		for i, child := range node.Children {
			children[i] = ResolveSyntax(child)
		}
		result := &bones.Node{Kind: bones.ListNode, Children: children, Loc: node.Loc}
		if node.DottedTail != nil {
			result.DottedTail = ResolveSyntax(node.DottedTail)
		}
		return result
	default:
		return node
	}
}
