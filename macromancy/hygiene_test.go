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
