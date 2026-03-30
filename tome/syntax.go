package tome

import (
	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
	"github.com/archevel/ghoul/macromancy"
	"github.com/archevel/ghoul/sarcophagus"
)

// stripMarks recursively removes hygiene marks from Node trees.
func stripMarks(node *e.Node) *e.Node {
	if node == nil || node.IsNil() {
		return node
	}
	switch node.Kind {
	case e.SyntaxObjectNode:
		if node.Quoted != nil {
			return stripMarks(node.Quoted)
		}
		return e.Nil
	case e.IdentifierNode:
		if len(node.Marks) > 0 {
			return e.IdentNode(node.Name)
		}
		return node
	case e.ListNode:
		children := make([]*e.Node, len(node.Children))
		for i, child := range node.Children {
			children[i] = stripMarks(child)
		}
		return &e.Node{Kind: e.ListNode, Children: children, Loc: node.Loc, DottedTail: node.DottedTail}
	default:
		return node
	}
}

func registerSyntax(env *ev.Environment) {
	env.Register("syntax->datum", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		node := args[0]
		if node.Kind == e.SyntaxObjectNode && node.Quoted != nil {
			return node.Quoted, nil
		}
		if node.Kind == e.IdentifierNode && len(node.Marks) > 0 {
			return e.IdentNode(node.Name), nil
		}
		return node, nil
	})

	env.Register("datum->syntax", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		ctx := args[0]
		datum := args[1]
		marks := macromancy.NewMarkSet()
		if ctx.Kind == e.SyntaxObjectNode {
			marks = macromancy.MarkSet(ctx.Marks)
		}
		return macromancy.WrapSyntax(datum, marks), nil
	})

	env.Register("identifier?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		node := args[0]
		if node.Kind == e.SyntaxObjectNode && node.Quoted != nil {
			return e.BoolNode(node.Quoted.Kind == e.IdentifierNode), nil
		}
		return e.BoolNode(node.Kind == e.IdentifierNode), nil
	})

	env.Register("syntax-match?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		// (syntax-match? expr pattern literals)
		// Returns an association list of bindings or #f.
		// Both expr and pattern are stripped of hygiene marks before matching.
		expr := stripMarks(args[0])
		pattern := stripMarks(args[1])

		// Build literals map from the literals list
		literals := map[string]bool{}
		litNode := args[2]
		if litNode.Kind == e.ListNode {
			for _, child := range litNode.Children {
				if name := child.IdentName(); name != "" {
					literals[name] = true
				}
			}
		}

		patternVars := macromancy.ExtractPatternVars(pattern, literals)
		ellipsisVars := macromancy.ExtractEllipsisVars(pattern, literals)
		macro := macromancy.Macro{
			Pattern:      pattern,
			PatternVars:  patternVars,
			EllipsisVars: ellipsisVars,
			Literals:     literals,
		}

		ok, alist := macro.MatchAndBind(expr)
		if !ok {
			return e.BoolNode(false), nil
		}
		return alist, nil
	})

	// Mummy conversion functions
	wrapMummyConv := func(fn sarcophagus.NodeConversionFunc) func([]*e.Node, *ev.Evaluator) (*e.Node, error) {
		return func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
			return fn(args, evaluator)
		}
	}
	env.Register("bytes", wrapMummyConv(sarcophagus.BytesConvNode))
	env.Register("string-from-bytes", wrapMummyConv(sarcophagus.StringFromBytesNode))
	env.Register("int-slice", wrapMummyConv(sarcophagus.IntSliceNode))
	env.Register("float-slice", wrapMummyConv(sarcophagus.FloatSliceNode))
	env.Register("go-nil", wrapMummyConv(sarcophagus.GoNilNode))
}
