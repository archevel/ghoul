package macromancy

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/parser"
)

func parseExpr(t *testing.T, code string) e.Expr {
	t.Helper()
	res, parsed := parser.Parse(strings.NewReader(code))
	if res != 0 {
		t.Fatalf("failed to parse: %s", code)
	}
	return parsed.Expressions.First()
}

func TestExpandHygienicMarksTemplateIdentifiers(t *testing.T) {
	// Macro: (swap x y) -> (begin (define tmp x) (set! x y) (set! y tmp))
	pattern := parseExpr(t, "(swap x y)")
	body := parseExpr(t, "(begin (define tmp x) (set! x y) (set! y tmp))")
	patternVars := ExtractPatternVars(pattern)

	macro := Macro{Pattern: pattern, Body: body, PatternVars: patternVars}

	// Match against (swap a b)
	code := parseExpr(t, "(swap a b)")
	ok, bound := macro.Matches(code)
	if !ok {
		t.Fatal("macro should match")
	}

	var mark Mark = 1
	result := macro.ExpandHygienic(bound, mark)

	// The result should be (begin (define tmp x) (set! x y) (set! y tmp))
	// with "a" for x, "b" for y, and tmp/begin/define/set! as ScopedIdentifiers with mark 1
	resultList, ok2 := result.(e.List)
	if !ok2 {
		t.Fatalf("expected List result, got %T", result)
	}

	// Check that "begin" has mark 1 (it's a template identifier, not a pattern var)
	beginExpr := resultList.First()
	beginSI, ok3 := beginExpr.(e.ScopedIdentifier)
	if !ok3 {
		t.Fatalf("expected 'begin' to be ScopedIdentifier, got %T (%s)", beginExpr, beginExpr.Repr())
	}
	if !beginSI.Marks[mark] {
		t.Error("expected 'begin' to have mark 1")
	}
	if beginSI.Name != e.Identifier("begin") {
		t.Errorf("expected name 'begin', got '%s'", beginSI.Name)
	}

	// Check that "a" (substituted for x) is a plain Identifier, not marked
	// Navigate: (begin (define tmp a) ...)
	tail1, _ := resultList.Tail()
	defineExpr := tail1.First().(e.List) // (define tmp a)
	defineTail, _ := defineExpr.Tail()
	tmpAndA, _ := defineTail.Tail()
	aExpr := tmpAndA.First()
	if _, isSI := aExpr.(e.ScopedIdentifier); isSI {
		t.Error("'a' (from user input) should NOT be a ScopedIdentifier")
	}
	if aExpr != e.Identifier("a") {
		t.Errorf("expected 'a', got %s (%T)", aExpr.Repr(), aExpr)
	}

	// Check that "tmp" IS a ScopedIdentifier with mark 1
	tmpExpr := defineTail.First()
	tmpSI, ok4 := tmpExpr.(e.ScopedIdentifier)
	if !ok4 {
		t.Fatalf("expected 'tmp' to be ScopedIdentifier, got %T (%s)", tmpExpr, tmpExpr.Repr())
	}
	if !tmpSI.Marks[mark] {
		t.Error("expected 'tmp' to have mark 1")
	}
}

func TestExpandHygienicWithEllipsisSplicesBindings(t *testing.T) {
	// Pattern: (my-begin x ...) with body (begin x ...)
	// Matched against (my-begin a b c)
	// Should expand to (begin a b c), not (begin a . (b c))
	pattern := parseExpr(t, "(my-begin x ...)")
	body := parseExpr(t, "(begin x ...)")
	patternVars := ExtractPatternVars(pattern)

	macro := Macro{Pattern: pattern, Body: body, PatternVars: patternVars}

	code := parseExpr(t, "(my-begin 1 2 3)")
	ok, bound := macro.Matches(code)
	if !ok {
		t.Fatal("macro should match")
	}

	var mark Mark = 1
	result := macro.ExpandHygienic(bound, mark)

	resultList, ok2 := result.(e.List)
	if !ok2 {
		t.Fatalf("expected List result, got %T: %s", result, result.Repr())
	}

	// Should be (begin 1 2 3) — 4 elements
	count := 0
	for l := resultList; l != e.NIL; {
		count++
		tail, ok := l.Tail()
		if !ok {
			break
		}
		l = tail
	}
	if count != 4 {
		t.Errorf("expected 4 elements (begin 1 2 3), got %d: %s", count, result.Repr())
	}

	// Verify the elements: begin(marked), 1, 2, 3
	tail1, _ := resultList.Tail()
	if !tail1.First().Equiv(e.Integer(1)) {
		t.Errorf("expected second element to be 1, got %s", tail1.First().Repr())
	}
	tail2, _ := tail1.Tail()
	if !tail2.First().Equiv(e.Integer(2)) {
		t.Errorf("expected third element to be 2, got %s", tail2.First().Repr())
	}
	tail3, _ := tail2.Tail()
	if !tail3.First().Equiv(e.Integer(3)) {
		t.Errorf("expected fourth element to be 3, got %s", tail3.First().Repr())
	}
}

func TestMatchWithLiteralsRequiresExactMatch(t *testing.T) {
	pattern := parseExpr(t, "(test-lit x arrow y)")
	body := parseExpr(t, "(+ x y)")
	literals := map[e.Identifier]bool{e.Identifier("arrow"): true}
	patternVars := ExtractPatternVarsWithLiterals(pattern, literals)

	macro := Macro{Pattern: pattern, Body: body, PatternVars: patternVars, Literals: literals}

	// Should match when literal is in the right position
	code1 := parseExpr(t, "(test-lit 3 arrow 4)")
	ok1, bound1 := macro.Matches(code1)
	if !ok1 {
		t.Error("should match when literal 'arrow' is present")
	}
	if bound1[e.Identifier("arrow")] != nil {
		t.Error("literal 'arrow' should NOT be bound as a variable")
	}
	if !bound1[e.Identifier("x")].Equiv(e.Integer(3)) {
		t.Error("x should be bound to 3")
	}

	// Should NOT match when a different identifier is in the literal position
	code2 := parseExpr(t, "(test-lit 3 blah 4)")
	ok2, _ := macro.Matches(code2)
	if ok2 {
		t.Error("should NOT match when 'blah' is where literal 'arrow' is expected")
	}
}

func TestExpandHygienicWithDefinitionBindingsSkipsKnownIdentifiers(t *testing.T) {
	pattern := parseExpr(t, "(mac x)")
	body := parseExpr(t, "(+ x 1)")
	patternVars := ExtractPatternVars(pattern)

	code := parseExpr(t, "(mac 5)")
	macro := Macro{Pattern: pattern, Body: body, PatternVars: patternVars}
	ok, bound := macro.Matches(code)
	if !ok {
		t.Fatal("should match")
	}

	defBindings := map[e.Identifier]bool{e.Identifier("+"): true}
	result := ExpandHygienicWithDefinitionBindings(body, bound, 1, patternVars, defBindings)

	list := result.(e.List)
	plus := list.First()
	if _, isSI := plus.(e.ScopedIdentifier); isSI {
		t.Error("'+' is a definition binding, should remain plain Identifier")
	}
	if id, ok := plus.(e.Identifier); !ok || id != e.Identifier("+") {
		t.Errorf("expected plain Identifier '+', got %T %s", plus, plus.Repr())
	}
}

func TestExpandHygienicWithScopedIdentifierInBody(t *testing.T) {
	// When the body contains a ScopedIdentifier that's a bound pattern var,
	// it should be substituted
	body := e.Cons(
		e.ScopedIdentifier{Name: "x", Marks: map[uint64]bool{1: true}},
		e.NIL,
	)
	bound := bindings{e.Identifier("x"): e.Integer(42)}
	patternVars := map[e.Identifier]bool{e.Identifier("x"): true}

	result := ExpandHygienicWithDefinitionBindings(body, bound, 2, patternVars, nil)
	list := result.(e.List)
	if !list.First().Equiv(e.Integer(42)) {
		t.Errorf("ScopedIdentifier 'x' should be substituted with 42, got %s", list.First().Repr())
	}
}

func TestExpandHygienicScopedIdentifierAccumulatesMarks(t *testing.T) {
	// A ScopedIdentifier that's not bound should accumulate the expansion mark
	body := e.ScopedIdentifier{Name: "y", Marks: map[uint64]bool{1: true}}
	result := ExpandHygienicWithDefinitionBindings(body, bindings{}, 2, nil, nil)

	si, ok := result.(e.ScopedIdentifier)
	if !ok {
		t.Fatalf("expected ScopedIdentifier, got %T", result)
	}
	if !si.Marks[1] || !si.Marks[2] {
		t.Error("expected both marks 1 and 2")
	}
}

func TestExpandHygienicScopedIdentifierDefinitionBinding(t *testing.T) {
	body := e.ScopedIdentifier{Name: "begin", Marks: map[uint64]bool{1: true}}
	defBindings := map[e.Identifier]bool{e.Identifier("begin"): true}
	result := ExpandHygienicWithDefinitionBindings(body, bindings{}, 2, nil, defBindings)

	si, ok := result.(e.ScopedIdentifier)
	if !ok {
		t.Fatalf("expected ScopedIdentifier, got %T", result)
	}
	if si.Marks[2] {
		t.Error("definition binding should not get additional marks")
	}
}

func TestExpandHygienicEllipsisNotBound(t *testing.T) {
	// When ... isn't in the bindings, it should be treated as a regular
	// template identifier and get marked
	body := parseExpr(t, "(foo ...)")
	result := ExpandHygienicWithDefinitionBindings(body, bindings{}, 1, nil, nil)

	list := result.(e.List)
	second := list.Second()
	// ... should be marked since it's not bound
	secondList, ok := second.(e.List)
	if !ok {
		t.Fatalf("expected list for rest, got %T: %s", second, second.Repr())
	}
	ellipsis := secondList.First()
	si, isSI := ellipsis.(e.ScopedIdentifier)
	if !isSI {
		t.Fatalf("expected ScopedIdentifier for unbound ..., got %T", ellipsis)
	}
	if si.Name != e.Identifier("...") {
		t.Errorf("expected name '...', got '%s'", si.Name)
	}
}

func TestExpandHygienicEllipsisEmptyList(t *testing.T) {
	// When ... is bound to NIL, splicing should produce nothing extra
	body := parseExpr(t, "(begin x ...)")
	bound := bindings{
		e.Identifier("x"):   e.Integer(1),
		e.Identifier("..."): e.NIL,
	}

	result := ExpandHygienicWithDefinitionBindings(body, bound, 1, nil, nil)
	list := result.(e.List)

	// Should be (begin 1) — just 2 elements, no extra from empty ellipsis
	count := 0
	for l := list; l != e.NIL; {
		count++
		tail, ok := l.Tail()
		if !ok {
			break
		}
		l = tail
	}
	if count != 2 {
		t.Errorf("expected 2 elements (begin 1), got %d: %s", count, result.Repr())
	}
}

func TestExpandHygienicUserVarNamedTmpDoesNotConflict(t *testing.T) {
	// Macro: (swap x y) -> (begin (define tmp x) (set! x y) (set! y tmp))
	pattern := parseExpr(t, "(swap x y)")
	body := parseExpr(t, "(begin (define tmp x) (set! x y) (set! y tmp))")
	patternVars := ExtractPatternVars(pattern)

	macro := Macro{Pattern: pattern, Body: body, PatternVars: patternVars}

	// User calls (swap tmp other) — their "tmp" should pass through without marks
	code := parseExpr(t, "(swap tmp other)")
	ok, bound := macro.Matches(code)
	if !ok {
		t.Fatal("macro should match")
	}

	var mark Mark = 1
	result := macro.ExpandHygienic(bound, mark)

	// In the expansion, the user's "tmp" (bound to x) is plain Identifier("tmp")
	// The macro's "tmp" is ScopedIdentifier{Name:"tmp", Marks:{1:true}}
	// They should not be Equiv
	resultList := result.(e.List)
	tail1, _ := resultList.Tail()
	defineExpr := tail1.First().(e.List) // (define <macro-tmp> <user-tmp>)
	defineTail, _ := defineExpr.Tail()

	macroTmp := defineTail.First()       // macro's tmp
	tmpAndUserTmp, _ := defineTail.Tail()
	userTmp := tmpAndUserTmp.First()      // user's tmp (was bound to x)

	if macroTmp.Equiv(userTmp) {
		t.Error("macro's tmp and user's tmp should NOT be Equiv")
	}

	// User's tmp should be plain
	if _, isSI := userTmp.(e.ScopedIdentifier); isSI {
		t.Error("user's tmp should be plain Identifier")
	}
	// Macro's tmp should be scoped
	macroTmpSI, ok2 := macroTmp.(e.ScopedIdentifier)
	if !ok2 {
		t.Fatalf("macro's tmp should be ScopedIdentifier, got %T", macroTmp)
	}
	if macroTmpSI.Name != e.Identifier("tmp") {
		t.Errorf("macro's tmp name should be 'tmp', got '%s'", macroTmpSI.Name)
	}
}
