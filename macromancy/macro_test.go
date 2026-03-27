package macromancy

import (
	"testing"

	"github.com/archevel/ghoul/bones"
)

func n(name string) *bones.Node { return bones.IdentNode(name) }
func i(v int64) *bones.Node     { return bones.IntNode(v) }
func list(children ...*bones.Node) *bones.Node {
	return bones.NewListNode(children)
}

func TestNodeMatchSimple(t *testing.T) {
	// Pattern: (mac x)  Code: (mac 42)
	m := Macro{
		Pattern: list(n("mac"), n("x")),
		Body:    n("x"),
	}
	ok, bound := m.matches(list(n("mac"), i(42)))
	if !ok {
		t.Fatal("expected match")
	}
	if bound.vars["x"].IntVal != 42 {
		t.Errorf("expected x=42, got %s", bound.vars["x"].Repr())
	}
}

func TestNodeMatchTwoVars(t *testing.T) {
	// Pattern: (mac x y)  Code: (mac 1 2)
	m := Macro{
		Pattern: list(n("mac"), n("x"), n("y")),
		Body:    list(n("+"), n("x"), n("y")),
	}
	ok, bound := m.matches(list(n("mac"), i(1), i(2)))
	if !ok {
		t.Fatal("expected match")
	}
	if bound.vars["x"].IntVal != 1 || bound.vars["y"].IntVal != 2 {
		t.Error("wrong bindings")
	}
}

func TestNodeMatchFails(t *testing.T) {
	// Pattern: (mac x y)  Code: (mac 1) — too few args
	m := Macro{
		Pattern: list(n("mac"), n("x"), n("y")),
	}
	ok, _ := m.matches(list(n("mac"), i(1)))
	if ok {
		t.Error("expected no match — too few args")
	}
}

func TestNodeMatchWildcard(t *testing.T) {
	// Pattern: (mac _ x)  Code: (mac 1 2)
	m := Macro{
		Pattern: list(n("mac"), n("_"), n("x")),
	}
	ok, bound := m.matches(list(n("mac"), i(1), i(2)))
	if !ok {
		t.Fatal("expected match")
	}
	if _, has := bound.vars["_"]; has {
		t.Error("wildcard should not create binding")
	}
	if bound.vars["x"].IntVal != 2 {
		t.Errorf("expected x=2, got %s", bound.vars["x"].Repr())
	}
}

func TestNodeMatchLiteral(t *testing.T) {
	// Pattern: (mac => x) with => as literal
	m := Macro{
		Pattern:  list(n("mac"), n("=>"), n("x")),
		Literals: map[string]bool{"=>": true},
	}
	ok, _ := m.matches(list(n("mac"), n("=>"), i(42)))
	if !ok {
		t.Fatal("expected match with literal")
	}

	// Should fail if literal doesn't match
	ok2, _ := m.matches(list(n("mac"), n("foo"), i(42)))
	if ok2 {
		t.Error("expected no match — literal mismatch")
	}
}

func TestNodeMatchNestedList(t *testing.T) {
	// Pattern: (mac (x y))  Code: (mac (1 2))
	m := Macro{
		Pattern: list(n("mac"), list(n("x"), n("y"))),
	}
	ok, bound := m.matches(list(n("mac"), list(i(1), i(2))))
	if !ok {
		t.Fatal("expected match")
	}
	if bound.vars["x"].IntVal != 1 || bound.vars["y"].IntVal != 2 {
		t.Error("wrong bindings for nested match")
	}
}

func TestNodeMatchEllipsis(t *testing.T) {
	// Pattern: (mac x ...)  Code: (mac 1 2 3)
	m := Macro{
		Pattern: list(n("mac"), n("x"), n("...")),
	}
	ok, bound := m.matches(list(n("mac"), i(1), i(2), i(3)))
	if !ok {
		t.Fatal("expected match")
	}
	if len(bound.repeated["x"]) != 3 {
		t.Fatalf("expected 3 repeated bindings, got %d", len(bound.repeated["x"]))
	}
	if bound.repeated["x"][0].IntVal != 1 || bound.repeated["x"][2].IntVal != 3 {
		t.Error("wrong repeated bindings")
	}
}

func TestNodeMatchEllipsisZeroRepetitions(t *testing.T) {
	// Pattern: (mac x ...)  Code: (mac)
	m := Macro{
		Pattern: list(n("mac"), n("x"), n("...")),
	}
	ok, bound := m.matches(list(n("mac")))
	if !ok {
		t.Fatal("expected match with zero repetitions")
	}
	if len(bound.repeated["x"]) != 0 {
		t.Errorf("expected 0 repeated bindings, got %d", len(bound.repeated["x"]))
	}
}

func TestNodeMatchEllipsisWithTailPattern(t *testing.T) {
	// Pattern: (mac x ... y)  Code: (mac 1 2 3 99)
	// x should match [1 2 3], y should match 99
	m := Macro{
		Pattern: list(n("mac"), n("x"), n("..."), n("y")),
	}
	ok, bound := m.matches(list(n("mac"), i(1), i(2), i(3), i(99)))
	if !ok {
		t.Fatal("expected match")
	}
	if len(bound.repeated["x"]) != 3 {
		t.Fatalf("expected 3 repeated x, got %d", len(bound.repeated["x"]))
	}
	if bound.vars["y"].IntVal != 99 {
		t.Errorf("expected y=99, got %s", bound.vars["y"].Repr())
	}
}

func TestNodeMatchNestedEllipsis(t *testing.T) {
	// Pattern: (mac (var val) ...)  Code: (mac (x 1) (y 2))
	m := Macro{
		Pattern: list(n("mac"), list(n("var"), n("val")), n("...")),
	}
	ok, bound := m.matches(list(n("mac"), list(n("x"), i(1)), list(n("y"), i(2))))
	if !ok {
		t.Fatal("expected match")
	}
	if len(bound.repeated["var"]) != 2 || len(bound.repeated["val"]) != 2 {
		t.Fatalf("expected 2 repeated each, got var=%d val=%d",
			len(bound.repeated["var"]), len(bound.repeated["val"]))
	}
}

// --- Expansion tests ---

func TestNodeExpandSimple(t *testing.T) {
	bound := newBindings()
	bound.vars["x"] = i(42)
	body := n("x")
	result := expandHygienic(body, bound, 1, map[string]bool{"x": true}, nil)
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestNodeExpandHygienicMarking(t *testing.T) {
	bound := newBindings()
	body := n("y") // not a pattern var
	result := expandHygienic(body, bound, 1, map[string]bool{}, nil)
	if result.Kind != bones.IdentifierNode {
		t.Fatalf("expected IdentifierNode, got %d", result.Kind)
	}
	if !result.Marks[1] {
		t.Error("expected mark 1 on non-pattern-var identifier")
	}
}

func TestNodeExpandDefinitionBinding(t *testing.T) {
	bound := newBindings()
	body := n("+") // bound at definition site
	defBindings := map[string]bool{"+": true}
	result := expandHygienic(body, bound, 1, map[string]bool{}, defBindings)
	if result.Kind != bones.IdentifierNode || result.Name != "+" {
		t.Errorf("expected plain +, got %s", result.Repr())
	}
	if len(result.Marks) > 0 {
		t.Error("definition-bound identifier should not be marked")
	}
}

func TestNodeExpandEllipsis(t *testing.T) {
	// Body: (begin x ...)
	// Bound: x → [1, 2, 3] (repeated)
	bound := newBindings()
	bound.repeated["x"] = []*bones.Node{i(1), i(2), i(3)}
	body := list(n("begin"), n("x"), n("..."))
	defBindings := map[string]bool{"begin": true}
	result := expandHygienic(body, bound, 1, map[string]bool{"x": true}, defBindings)
	if result.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", result.Kind)
	}
	// Should be (begin 1 2 3)
	if len(result.Children) != 4 {
		t.Fatalf("expected 4 children (begin + 3 values), got %d: %s", len(result.Children), result.Repr())
	}
	if result.Children[1].IntVal != 1 || result.Children[2].IntVal != 2 || result.Children[3].IntVal != 3 {
		t.Errorf("expected (begin 1 2 3), got %s", result.Repr())
	}
}

func TestNodeExpandNestedEllipsis(t *testing.T) {
	// let macro: body is ((lambda (var ...) body ...) val ...)
	// Bound: var → [x, y], val → [1, 2], body → [expr]
	bound := newBindings()
	bound.repeated["var"] = []*bones.Node{n("x"), n("y")}
	bound.repeated["val"] = []*bones.Node{i(1), i(2)}
	bound.vars["body"] = list(n("+"), n("x"), n("y"))

	// Template: ((lambda (var ...) body) val ...)
	body := list(
		list(n("lambda"), list(n("var"), n("...")), n("body")),
		n("val"),
		n("..."),
	)
	defBindings := map[string]bool{"lambda": true}
	result := expandHygienic(body, bound, 1, map[string]bool{"var": true, "val": true, "body": true}, defBindings)

	// Should be ((lambda (x y) (+ x y)) 1 2)
	if result.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", result.Kind)
	}
	if len(result.Children) != 3 {
		t.Fatalf("expected 3 children, got %d: %s", len(result.Children), result.Repr())
	}
}

// --- Full transformer test ---

func TestBuildSyntaxRulesTransformer(t *testing.T) {
	// (syntax-rules () ((my-add x y) (+ x y)))
	syntaxRules := list(
		n("syntax-rules"),
		list(), // empty literals
		list(
			list(n("my-add"), n("x"), n("y")),
			list(n("+"), n("x"), n("y")),
		),
	)

	defBindings := map[string]bool{"+": true}
	st, err := BuildSyntaxRulesTransformer("my-add", syntaxRules, defBindings)
	if err != nil {
		t.Fatal(err)
	}

	// Transform (my-add 1 2)
	code := list(n("my-add"), i(1), i(2))
	result, err := st.Transform(code, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Should be (+ 1 2)
	if result.Kind != bones.ListNode || len(result.Children) != 3 {
		t.Fatalf("expected (+ 1 2), got %s", result.Repr())
	}
	if result.Children[0].Name != "+" {
		t.Errorf("expected +, got %s", result.Children[0].Repr())
	}
	if result.Children[1].IntVal != 1 || result.Children[2].IntVal != 2 {
		t.Errorf("expected 1 2, got %s", result.Repr())
	}
}

func TestBuildNodeSyntaxRulesWithEllipsis(t *testing.T) {
	// (syntax-rules () ((my-list x ...) (list x ...)))
	syntaxRules := list(
		n("syntax-rules"),
		list(),
		list(
			list(n("my-list"), n("x"), n("...")),
			list(n("list"), n("x"), n("...")),
		),
	)

	defBindings := map[string]bool{"list": true}
	st, err := BuildSyntaxRulesTransformer("my-list", syntaxRules, defBindings)
	if err != nil {
		t.Fatal(err)
	}

	// Transform (my-list 1 2 3)
	code := list(n("my-list"), i(1), i(2), i(3))
	result, err := st.Transform(code, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Should be (list 1 2 3)
	if result.Kind != bones.ListNode || len(result.Children) != 4 {
		t.Fatalf("expected (list 1 2 3), got %s", result.Repr())
	}
}

func TestExtractNodePatternVars(t *testing.T) {
	// (mac x y)
	pat := list(n("mac"), n("x"), n("y"))
	vars := extractPatternVars(pat, nil)
	if !vars["x"] || !vars["y"] {
		t.Errorf("expected x and y in pattern vars, got %v", vars)
	}
	if vars["mac"] {
		t.Error("macro name should not be a pattern var")
	}
}

func TestExtractNodeEllipsisVars(t *testing.T) {
	// (mac x y ...)
	pat := list(n("mac"), n("x"), n("y"), n("..."))
	vars := extractEllipsisVars(pat, nil)
	if !vars["y"] {
		t.Error("expected y in ellipsis vars")
	}
	if vars["x"] {
		t.Error("x should not be in ellipsis vars")
	}
}

// --- Additional coverage tests ---

func TestNodeMatchPatternNameMismatch(t *testing.T) {
	// Pattern head is "mac" but code head is "other" — should still match
	// because matches() skips the first child (macro name) on both sides.
	m := Macro{
		Pattern: list(n("mac"), n("x")),
		Body:    n("x"),
	}
	ok, bound := m.matches(list(n("other"), i(7)))
	if !ok {
		t.Fatal("expected match — first child is skipped by matches()")
	}
	if bound.vars["x"].IntVal != 7 {
		t.Errorf("expected x=7, got %s", bound.vars["x"].Repr())
	}
}

func TestNodeMatchNonMatchingPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern *bones.Node
		code    *bones.Node
	}{
		{
			name:    "too many args",
			pattern: list(n("mac"), n("x")),
			code:    list(n("mac"), i(1), i(2)),
		},
		{
			name:    "nested list vs atom",
			pattern: list(n("mac"), list(n("x"), n("y"))),
			code:    list(n("mac"), i(1)),
		},
		{
			name:    "atom vs nested list",
			pattern: list(n("mac"), n("x"), n("y")),
			code:    list(n("mac"), list(i(1), i(2))),
		},
		{
			name:    "empty code",
			pattern: list(n("mac"), n("x")),
			code:    list(n("mac")),
		},
		{
			name:    "empty pattern non-empty code",
			pattern: list(n("mac")),
			code:    list(n("mac"), i(1)),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Macro{Pattern: tc.pattern}
			ok, _ := m.matches(tc.code)
			if ok {
				t.Errorf("expected no match for %s", tc.name)
			}
		})
	}
}

func TestNodeMatchNestedEllipsisFourBindings(t *testing.T) {
	// Pattern: (mac (a b) ...)  Code: (mac (w 1) (x 2) (y 3) (z 4))
	m := Macro{
		Pattern: list(n("mac"), list(n("a"), n("b")), n("...")),
	}
	code := list(n("mac"),
		list(n("w"), i(1)),
		list(n("x"), i(2)),
		list(n("y"), i(3)),
		list(n("z"), i(4)),
	)
	ok, bound := m.matches(code)
	if !ok {
		t.Fatal("expected match")
	}
	if len(bound.repeated["a"]) != 4 {
		t.Fatalf("expected 4 repeated a, got %d", len(bound.repeated["a"]))
	}
	if len(bound.repeated["b"]) != 4 {
		t.Fatalf("expected 4 repeated b, got %d", len(bound.repeated["b"]))
	}
	if bound.repeated["a"][0].Name != "w" || bound.repeated["a"][3].Name != "z" {
		t.Errorf("wrong a bindings: %s, %s", bound.repeated["a"][0].Repr(), bound.repeated["a"][3].Repr())
	}
	if bound.repeated["b"][0].IntVal != 1 || bound.repeated["b"][3].IntVal != 4 {
		t.Errorf("wrong b bindings: %s, %s", bound.repeated["b"][0].Repr(), bound.repeated["b"][3].Repr())
	}
}

func TestNodeMatchNestedEllipsisWithTrailingPattern(t *testing.T) {
	// Pattern: (mac (a b) ... c)  Code: (mac (x 1) (y 2) 99)
	m := Macro{
		Pattern: list(n("mac"), list(n("a"), n("b")), n("..."), n("c")),
	}
	code := list(n("mac"), list(n("x"), i(1)), list(n("y"), i(2)), i(99))
	ok, bound := m.matches(code)
	if !ok {
		t.Fatal("expected match")
	}
	if len(bound.repeated["a"]) != 2 {
		t.Fatalf("expected 2 repeated a, got %d", len(bound.repeated["a"]))
	}
	if len(bound.repeated["b"]) != 2 {
		t.Fatalf("expected 2 repeated b, got %d", len(bound.repeated["b"]))
	}
	if bound.vars["c"].IntVal != 99 {
		t.Errorf("expected c=99, got %s", bound.vars["c"].Repr())
	}
}

func TestNodeMatchWildcardVariousPositions(t *testing.T) {
	t.Run("wildcard at head position", func(t *testing.T) {
		// Pattern: (mac _ x)  Code: (mac (complex stuff) 42)
		m := Macro{
			Pattern: list(n("mac"), n("_"), n("x")),
		}
		ok, bound := m.matches(list(n("mac"), list(i(1), i(2)), i(42)))
		if !ok {
			t.Fatal("expected match")
		}
		if _, has := bound.vars["_"]; has {
			t.Error("wildcard should not create binding")
		}
		if bound.vars["x"].IntVal != 42 {
			t.Errorf("expected x=42, got %s", bound.vars["x"].Repr())
		}
	})

	t.Run("wildcard at tail position", func(t *testing.T) {
		// Pattern: (mac x _)  Code: (mac 42 anything)
		m := Macro{
			Pattern: list(n("mac"), n("x"), n("_")),
		}
		ok, bound := m.matches(list(n("mac"), i(42), n("anything")))
		if !ok {
			t.Fatal("expected match")
		}
		if _, has := bound.vars["_"]; has {
			t.Error("wildcard should not create binding")
		}
		if bound.vars["x"].IntVal != 42 {
			t.Errorf("expected x=42, got %s", bound.vars["x"].Repr())
		}
	})

	t.Run("multiple wildcards", func(t *testing.T) {
		// Pattern: (mac _ _ x)  Code: (mac a b 42)
		m := Macro{
			Pattern: list(n("mac"), n("_"), n("_"), n("x")),
		}
		ok, bound := m.matches(list(n("mac"), n("a"), n("b"), i(42)))
		if !ok {
			t.Fatal("expected match")
		}
		if _, has := bound.vars["_"]; has {
			t.Error("wildcard should not create binding")
		}
		if bound.vars["x"].IntVal != 42 {
			t.Errorf("expected x=42, got %s", bound.vars["x"].Repr())
		}
	})

	t.Run("wildcard matches list", func(t *testing.T) {
		// Pattern: (mac _)  Code: (mac (1 2 3))
		m := Macro{
			Pattern: list(n("mac"), n("_")),
		}
		ok, bound := m.matches(list(n("mac"), list(i(1), i(2), i(3))))
		if !ok {
			t.Fatal("expected match — wildcard should match a list")
		}
		if _, has := bound.vars["_"]; has {
			t.Error("wildcard should not create binding")
		}
	})
}

func TestBuildNodeSyntaxRulesMultipleClauses(t *testing.T) {
	// Two clauses: first matches (my-mac x y), second matches (my-mac x)
	syntaxRules := list(
		n("syntax-rules"),
		list(), // no literals
		list(
			list(n("my-mac"), n("x"), n("y")),
			list(n("+"), n("x"), n("y")),
		),
		list(
			list(n("my-mac"), n("x")),
			n("x"),
		),
	)

	defBindings := map[string]bool{"+": true}
	st, err := BuildSyntaxRulesTransformer("my-mac", syntaxRules, defBindings)
	if err != nil {
		t.Fatal(err)
	}

	// (my-mac 5) should NOT match first clause (needs 2 args), SHOULD match second
	code := list(n("my-mac"), i(5))
	result, err := st.Transform(code, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.IntVal != 5 {
		t.Errorf("expected 5, got %s", result.Repr())
	}

	// (my-mac 3 4) should match the first clause
	code2 := list(n("my-mac"), i(3), i(4))
	result2, err := st.Transform(code2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result2.Kind != bones.ListNode || len(result2.Children) != 3 {
		t.Fatalf("expected (+ 3 4), got %s", result2.Repr())
	}
	if result2.Children[1].IntVal != 3 || result2.Children[2].IntVal != 4 {
		t.Errorf("expected (+ 3 4), got %s", result2.Repr())
	}
}

func TestNodeMatchLiteralFailure(t *testing.T) {
	// Pattern: (mac => x) with => literal. Code uses different identifier.
	m := Macro{
		Pattern:  list(n("mac"), n("=>"), n("x")),
		Literals: map[string]bool{"=>": true},
	}

	// Non-identifier in literal position should fail
	ok, _ := m.matches(list(n("mac"), i(42), i(7)))
	if ok {
		t.Error("expected no match — literal position has integer, not =>")
	}

	// Wrong identifier should fail
	ok2, _ := m.matches(list(n("mac"), n("->"), i(7)))
	if ok2 {
		t.Error("expected no match — literal -> does not match =>")
	}
}

func TestNodeExpandEllipsisEmptyRepeatedBindings(t *testing.T) {
	// Body: (begin x ...) with x repeated 0 times => (begin)
	bound := newBindings()
	bound.repeated["x"] = []*bones.Node{}
	body := list(n("begin"), n("x"), n("..."))
	defBindings := map[string]bool{"begin": true}
	result := expandHygienic(body, bound, 1, map[string]bool{"x": true}, defBindings)
	if result.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", result.Kind)
	}
	// Should be (begin) — only the "begin" element, no repeated values
	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child (begin only), got %d: %s", len(result.Children), result.Repr())
	}
	if result.Children[0].Name != "begin" {
		t.Errorf("expected begin, got %s", result.Children[0].Repr())
	}
}

func TestNodeExpandScopedIdentAccumulatesMarks(t *testing.T) {
	// A ScopedIdentifier in the body that already has marks should accumulate the new mark
	bound := newBindings()
	body := bones.ScopedIdentNode("y", map[uint64]bool{5: true})
	result := expandHygienic(body, bound, 10, map[string]bool{}, nil)
	if result.Kind != bones.IdentifierNode {
		t.Fatalf("expected IdentifierNode, got %d", result.Kind)
	}
	if !result.Marks[5] {
		t.Error("expected existing mark 5 to be preserved")
	}
	if !result.Marks[10] {
		t.Error("expected new mark 10 to be added")
	}
	if result.Name != "y" {
		t.Errorf("expected name y, got %s", result.Name)
	}
}

func TestNodeMatchWithScopedIdentifierAtHead(t *testing.T) {
	// Pattern with a ScopedIdentifier as the macro name at head position
	scopedHead := bones.ScopedIdentNode("mac", map[uint64]bool{1: true})
	m := Macro{
		Pattern: list(scopedHead, n("x")),
		Body:    n("x"),
	}
	// Code also uses a scoped identifier at head
	codeHead := bones.ScopedIdentNode("mac", map[uint64]bool{1: true})
	ok, bound := m.matches(list(codeHead, i(42)))
	if !ok {
		t.Fatal("expected match — head is skipped by matches()")
	}
	if bound.vars["x"].IntVal != 42 {
		t.Errorf("expected x=42, got %s", bound.vars["x"].Repr())
	}
}

func TestMatchAndBindReturnsDottedPairs(t *testing.T) {
	// Pattern: (mac x y)  Code: (mac 1 2)
	m := Macro{
		Pattern:     list(n("mac"), n("x"), n("y")),
		Body:        list(n("+"), n("x"), n("y")),
		PatternVars: map[string]bool{"x": true, "y": true},
	}
	ok, assocList := m.MatchAndBind(list(n("mac"), i(1), i(2)))
	if !ok {
		t.Fatal("expected match")
	}
	if assocList.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", assocList.Kind)
	}

	// Verify we got dotted pairs (name . value)
	found := map[string]int64{}
	for _, pair := range assocList.Children {
		if pair.Kind != bones.ListNode {
			t.Fatalf("expected pair to be ListNode, got %d", pair.Kind)
		}
		if len(pair.Children) != 1 {
			t.Fatalf("expected 1 child in dotted pair, got %d", len(pair.Children))
		}
		if pair.DottedTail == nil {
			t.Fatal("expected dotted tail to be non-nil")
		}
		name := pair.Children[0].Name
		found[name] = pair.DottedTail.IntVal
	}

	if found["x"] != 1 {
		t.Errorf("expected x=1, got %d", found["x"])
	}
	if found["y"] != 2 {
		t.Errorf("expected y=2, got %d", found["y"])
	}
}

func TestMatchAndBindWithRepeatedVars(t *testing.T) {
	// Pattern: (mac x ...)  Code: (mac 1 2 3)
	m := Macro{
		Pattern:      list(n("mac"), n("x"), n("...")),
		Body:         list(n("begin"), n("x"), n("...")),
		PatternVars:  map[string]bool{"x": true},
		EllipsisVars: map[string]bool{"x": true},
	}
	ok, assocList := m.MatchAndBind(list(n("mac"), i(1), i(2), i(3)))
	if !ok {
		t.Fatal("expected match")
	}
	if assocList.Kind != bones.ListNode {
		t.Fatalf("expected ListNode, got %d", assocList.Kind)
	}

	// Should have one dotted pair for "x" with a list as the tail
	if len(assocList.Children) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(assocList.Children))
	}
	pair := assocList.Children[0]
	if pair.Children[0].Name != "x" {
		t.Errorf("expected pair name x, got %s", pair.Children[0].Name)
	}
	if pair.DottedTail == nil || pair.DottedTail.Kind != bones.ListNode {
		t.Fatal("expected dotted tail to be a list of repeated values")
	}
	if len(pair.DottedTail.Children) != 3 {
		t.Fatalf("expected 3 repeated values, got %d", len(pair.DottedTail.Children))
	}
}

func TestMatchAndBindNoMatch(t *testing.T) {
	// Pattern: (mac x y)  Code: (mac 1) — too few args
	m := Macro{
		Pattern: list(n("mac"), n("x"), n("y")),
	}
	ok, result := m.MatchAndBind(list(n("mac"), i(1)))
	if ok {
		t.Error("expected no match")
	}
	if result != nil {
		t.Error("expected nil result on no match")
	}
}
