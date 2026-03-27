package macromancy

import (
	"fmt"

	"github.com/archevel/ghoul/bones"
)

// bindings holds variable bindings during pattern matching.
type bindings struct {
	vars     map[string]*bones.Node
	repeated map[string][]*bones.Node
}

func newBindings() bindings {
	return bindings{vars: map[string]*bones.Node{}, repeated: map[string][]*bones.Node{}}
}

// Macro is a single pattern/template rule operating on *bones.Node.
type Macro struct {
	Pattern      *bones.Node
	Body         *bones.Node
	PatternVars  map[string]bool
	EllipsisVars map[string]bool
	Literals     map[string]bool
}

// SyntaxTransformer holds a pattern-based macro transformer for Node trees.
type SyntaxTransformer struct {
	Transform func(code *bones.Node, mark Mark) (*bones.Node, error)
}

// BuildSyntaxRulesTransformer creates a SyntaxTransformer from a
// syntax-rules form expressed as a *bones.Node tree.
func BuildSyntaxRulesTransformer(name string, syntaxRules *bones.Node, definitionBindings map[string]bool) (SyntaxTransformer, error) {
	macros, err := extractMacros(name, syntaxRules)
	if err != nil {
		return SyntaxTransformer{}, err
	}

	return SyntaxTransformer{
		Transform: func(code *bones.Node, mark Mark) (*bones.Node, error) {
			for _, m := range macros {
				if ok, bound := m.matches(code); ok {
					return expandHygienic(m.Body, bound, mark, m.PatternVars, definitionBindings), nil
				}
			}
			return nil, fmt.Errorf("no matching pattern for %s", code.Repr())
		},
	}, nil
}

func extractMacros(name string, syntaxRules *bones.Node) ([]Macro, error) {
	if syntaxRules.Kind != bones.ListNode || len(syntaxRules.Children) < 3 {
		return nil, fmt.Errorf("invalid syntax-rules: expected (syntax-rules (literals...) rules...)")
	}

	// syntaxRules.Children[0] = "syntax-rules"
	// syntaxRules.Children[1] = literals list
	// syntaxRules.Children[2:] = rules

	literals, err := extractLiterals(syntaxRules.Children[1])
	if err != nil {
		return nil, err
	}

	var macros []Macro
	for i := 2; i < len(syntaxRules.Children); i++ {
		rule := syntaxRules.Children[i]
		if rule.Kind != bones.ListNode || len(rule.Children) < 2 {
			return nil, fmt.Errorf("invalid rule definition: expected list for rule, got %s at position %d", bones.NodeTypeName(rule), i-2)
		}
		pat := rule.Children[0]
		body := rule.Children[1]
		macros = append(macros, Macro{
			Pattern:      pat,
			Body:         body,
			PatternVars:  extractPatternVars(pat, literals),
			EllipsisVars: extractEllipsisVars(pat, literals),
			Literals:     literals,
		})
	}
	return macros, nil
}

func extractLiterals(litNode *bones.Node) (map[string]bool, error) {
	literals := map[string]bool{}
	if litNode.IsNil() {
		return literals, nil
	}
	if litNode.Kind != bones.ListNode {
		return nil, fmt.Errorf("invalid syntax-rules: literals must be a list, got %s", litNode.Repr())
	}
	for _, child := range litNode.Children {
		name := child.IdentName()
		if name == "" {
			return nil, fmt.Errorf("invalid literals list: expected identifier, got %s", child.Repr())
		}
		literals[name] = true
	}
	return literals, nil
}

// --- Pattern matching ---

func (m Macro) matches(code *bones.Node) (bool, bindings) {
	if m.Pattern.Kind != bones.ListNode || code.Kind != bones.ListNode {
		return false, bindings{}
	}
	if len(m.Pattern.Children) == 0 || len(code.Children) == 0 {
		return false, bindings{}
	}
	// First child is the macro name — skip it, match the rest
	patChildren := m.Pattern.Children[1:]
	codeChildren := code.Children[1:]
	return matchChildren(patChildren, codeChildren, newBindings(), m.Literals)
}

func matchChildren(pattern []*bones.Node, code []*bones.Node, bound bindings, literals map[string]bool) (bool, bindings) {
	pi, ci := 0, 0

	for pi < len(pattern) {
		pat := pattern[pi]

		// Check if next pattern element is ellipsis
		if pi+1 < len(pattern) && isEllipsis(pattern[pi+1]) {
			// Count how many patterns follow the ellipsis
			tailPatterns := pattern[pi+2:]
			tailCount := len(tailPatterns)

			// Repeated elements = code elements minus those needed for tail
			repeatedCount := len(code) - ci - tailCount
			if repeatedCount < 0 {
				repeatedCount = 0
			}

			// Collect vars in subpattern
			subVars := map[string]bool{}
			collectIdentifiers(pat, subVars, literals)
			for v := range subVars {
				if bound.repeated == nil {
					bound.repeated = map[string][]*bones.Node{}
				}
				bound.repeated[v] = []*bones.Node{}
			}

			// Match each repeated element
			for i := 0; i < repeatedCount; i++ {
				if ci >= len(code) {
					break
				}
				localBound := newBindings()
				ok, localBound := matchExpr(pat, code[ci], localBound, literals)
				if !ok {
					return false, bindings{}
				}
				for v := range subVars {
					if val, exists := localBound.vars[v]; exists {
						bound.repeated[v] = append(bound.repeated[v], val)
					}
				}
				ci++
			}

			// Skip the ellipsis in pattern
			pi += 2
			continue
		}

		// Regular (non-ellipsis) match
		if ci >= len(code) {
			return false, bindings{}
		}
		var ok bool
		ok, bound = matchExpr(pat, code[ci], bound, literals)
		if !ok {
			return false, bindings{}
		}
		pi++
		ci++
	}

	// All pattern elements consumed — code should also be consumed
	if ci < len(code) {
		return false, bindings{}
	}
	return true, bound
}

func isEmptyList(n *bones.Node) bool {
	return n.IsNil() || (n.Kind == bones.ListNode && len(n.Children) == 0)
}

func matchExpr(pattern *bones.Node, code *bones.Node, bound bindings, literals map[string]bool) (bool, bindings) {
	// Both empty lists (Nil or ListNode with no children)
	if isEmptyList(pattern) && isEmptyList(code) {
		return true, bound
	}

	// Wildcard
	if pattern.Kind == bones.IdentifierNode && pattern.Name == "_" {
		return true, bound
	}

	// Pattern variable or literal
	if pattern.Kind == bones.IdentifierNode {
		name := pattern.Name
		// Literal check
		if literals != nil && literals[name] {
			codeName := code.IdentName()
			if codeName == name {
				return true, bound
			}
			return false, bindings{}
		}
		// Variable binding
		if existing, present := bound.vars[name]; present {
			if !existing.Equiv(code) {
				return false, bound
			}
			return true, bound
		}
		bound.vars[name] = code
		return true, bound
	}

	// List pattern vs list code (including Nil as empty list)
	if pattern.Kind == bones.ListNode {
		var codeChildren []*bones.Node
		if code.Kind == bones.ListNode {
			codeChildren = code.Children
		} else if isEmptyList(code) {
			codeChildren = nil
		} else if len(pattern.Children) == 1 {
			// Single-element pattern vs non-list code
			return matchExpr(pattern.Children[0], code, bound, literals)
		} else {
			return false, bindings{}
		}
		return matchChildren(pattern.Children, codeChildren, bound, literals)
	}

	// Literal value match
	if pattern.Equiv(code) {
		return true, bound
	}

	return false, bindings{}
}

func isEllipsis(n *bones.Node) bool {
	return n.Kind == bones.IdentifierNode && n.Name == "..."
}

func collectIdentifiers(node *bones.Node, vars map[string]bool, literals map[string]bool) {
	if node.Kind == bones.IdentifierNode {
		name := node.Name
		if name != "..." && name != "_" && (literals == nil || !literals[name]) {
			vars[name] = true
		}
		return
	}
	if node.Kind == bones.ListNode {
		for _, child := range node.Children {
			collectIdentifiers(child, vars, literals)
		}
	}
}

// --- Pattern variable extraction ---

func ExtractPatternVars(pattern *bones.Node, literals map[string]bool) map[string]bool {
	return extractPatternVars(pattern, literals)
}

func ExtractEllipsisVars(pattern *bones.Node, literals map[string]bool) map[string]bool {
	return extractEllipsisVars(pattern, literals)
}

func extractPatternVars(pattern *bones.Node, literals map[string]bool) map[string]bool {
	vars := map[string]bool{}
	if pattern.Kind != bones.ListNode || len(pattern.Children) < 2 {
		return vars
	}
	// Skip the macro name (first child)
	for _, child := range pattern.Children[1:] {
		collectIdentifiers(child, vars, literals)
	}
	return vars
}

func extractEllipsisVars(pattern *bones.Node, literals map[string]bool) map[string]bool {
	vars := map[string]bool{}
	if pattern.Kind != bones.ListNode || len(pattern.Children) < 2 {
		return vars
	}
	collectEllipsisVarsFromChildren(pattern.Children[1:], vars, literals)
	return vars
}

func collectEllipsisVarsFromChildren(children []*bones.Node, vars map[string]bool, literals map[string]bool) {
	for i := 0; i < len(children); i++ {
		if i+1 < len(children) && isEllipsis(children[i+1]) {
			collectIdentifiers(children[i], vars, literals)
			i++ // skip the ellipsis
			continue
		}
		if children[i].Kind == bones.ListNode {
			collectEllipsisVarsFromChildren(children[i].Children, vars, literals)
		}
	}
}

// --- Hygienic expansion ---

func expandHygienic(body *bones.Node, bound bindings, mark Mark, patternVars map[string]bool, definitionBindings map[string]bool) *bones.Node {
	return expandHygienicImpl(body, bound, mark, patternVars, definitionBindings)
}

func expandHygienicImpl(node *bones.Node, bound bindings, mark Mark, patternVars map[string]bool, defBindings map[string]bool) *bones.Node {
	if node == nil || node.IsNil() {
		return node
	}

	// Identifier: replace with binding or mark for hygiene
	if node.Kind == bones.IdentifierNode {
		name := node.Name
		if replacement, present := bound.vars[name]; present {
			return replacement
		}
		if defBindings != nil && defBindings[name] {
			return node
		}
		if len(node.Marks) > 0 {
			newMarks := copyMarks(node.Marks)
			newMarks[mark] = true
			return bones.ScopedIdentNode(name, newMarks)
		}
		return bones.ScopedIdentNode(name, map[uint64]bool{mark: true})
	}

	// List: handle ellipsis expansion
	if node.Kind == bones.ListNode {
		return expandList(node, bound, mark, patternVars, defBindings)
	}

	return node
}

func expandList(node *bones.Node, bound bindings, mark Mark, patternVars map[string]bool, defBindings map[string]bool) *bones.Node {
	var resultChildren []*bones.Node

	for i := 0; i < len(node.Children); i++ {
		child := node.Children[i]

		// Check if next child is ellipsis
		if i+1 < len(node.Children) && isEllipsis(node.Children[i+1]) {
			repeatedVars := findRepeatedVars(child, bound)
			if len(repeatedVars) > 0 {
				count := len(bound.repeated[repeatedVars[0]])
				for j := 0; j < count; j++ {
					iterBound := bindingsForIteration(bound, repeatedVars, j)
					expanded := expandHygienicImpl(child, iterBound, mark, patternVars, defBindings)
					resultChildren = append(resultChildren, expanded)
				}
				i++ // skip the ellipsis
				continue
			}
		}

		expanded := expandHygienicImpl(child, bound, mark, patternVars, defBindings)
		resultChildren = append(resultChildren, expanded)
	}

	result := bones.NewListNode(resultChildren)
	result.Loc = node.Loc
	return result
}

func findRepeatedVars(tmpl *bones.Node, bound bindings) []string {
	var result []string
	findRepeatedVarsWalk(tmpl, bound, &result)
	return result
}

func findRepeatedVarsWalk(node *bones.Node, bound bindings, result *[]string) {
	if node.Kind == bones.IdentifierNode {
		if _, hasRepeated := bound.repeated[node.Name]; hasRepeated {
			*result = append(*result, node.Name)
		}
		return
	}
	if node.Kind == bones.ListNode {
		for _, child := range node.Children {
			findRepeatedVarsWalk(child, bound, result)
		}
	}
}

func bindingsForIteration(bound bindings, repeatedVars []string, i int) bindings {
	iter := newBindings()
	for k, v := range bound.vars {
		iter.vars[k] = v
	}
	for _, v := range repeatedVars {
		if vals, ok := bound.repeated[v]; ok && i < len(vals) {
			iter.vars[v] = vals[i]
		}
	}
	for k, v := range bound.repeated {
		iter.repeated[k] = v
	}
	return iter
}

// --- MatchAndBind for Node trees ---

func (m Macro) MatchAndBind(code *bones.Node) (bool, *bones.Node) {
	ok, bound := m.matches(code)
	if !ok {
		return false, nil
	}

	// Build an association list of (name . value) dotted pairs
	var pairs []*bones.Node
	for name, val := range bound.vars {
		pair := &bones.Node{
			Kind:       bones.ListNode,
			Children:   []*bones.Node{bones.IdentNode(name)},
			DottedTail: val,
		}
		pairs = append(pairs, pair)
	}
	for name, vals := range bound.repeated {
		valList := bones.NewListNode(vals)
		pair := &bones.Node{
			Kind:       bones.ListNode,
			Children:   []*bones.Node{bones.IdentNode(name)},
			DottedTail: valList,
		}
		pairs = append(pairs, pair)
	}

	return true, bones.NewListNode(pairs)
}
