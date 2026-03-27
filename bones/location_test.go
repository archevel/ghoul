package bones

import (
	"os"
	"strings"
	"testing"
)

func TestSourcePositionLineAndColumn(t *testing.T) {
	pos := &SourcePosition{Ln: 3, Col: 7}
	if pos.Line() != 3 {
		t.Errorf("expected line 3, got %d", pos.Line())
	}
	if pos.Column() != 7 {
		t.Errorf("expected column 7, got %d", pos.Column())
	}
}

func TestSourcePositionString(t *testing.T) {
	pos := &SourcePosition{Ln: 1, Col: 10}
	if pos.String() != "1:10" {
		t.Errorf("expected '1:10', got '%s'", pos.String())
	}
}

func TestNodeCarriesLocation(t *testing.T) {
	loc := &SourcePosition{Ln: 3, Col: 5}
	// Use a value outside the int cache to avoid mutating a cached singleton
	node := IntNode(999)
	node.Loc = loc

	if node.Loc == nil {
		t.Fatal("expected Loc to be set")
	}
	if node.Loc.Line() != 3 || node.Loc.Column() != 5 {
		t.Errorf("expected 3:5, got %s", node.Loc.String())
	}
}

func TestNodeCreatesNilLocation(t *testing.T) {
	node := IntNode(999)
	if node.Loc != nil {
		t.Error("node constructor should create node with nil Loc by default")
	}
}

func TestSourcePositionWithFilename(t *testing.T) {
	filename := "test.ghoul"
	pos := &SourcePosition{Ln: 3, Col: 7, Filename: &filename}
	if pos.String() != "test.ghoul:3:7" {
		t.Errorf("expected 'test.ghoul:3:7', got '%s'", pos.String())
	}
}

func TestSourcePositionWithoutFilename(t *testing.T) {
	pos := &SourcePosition{Ln: 3, Col: 7}
	if pos.String() != "3:7" {
		t.Errorf("expected '3:7', got '%s'", pos.String())
	}
}

func TestSourceContextReadsFromFile(t *testing.T) {
	tmpFile := t.TempDir() + "/test.ghoul"
	os.WriteFile(tmpFile, []byte("line one\nline two\nline three\n"), 0644)

	pos := &SourcePosition{Ln: 2, Col: 1, Filename: &tmpFile}
	ctx := pos.SourceContext()
	if !strings.Contains(ctx, "line two") {
		t.Errorf("expected 'line two' in context, got '%s'", ctx)
	}
}

func TestSourceContextReturnsEmptyWhenNoFilename(t *testing.T) {
	pos := &SourcePosition{Ln: 1, Col: 1}
	if pos.SourceContext() != "" {
		t.Error("expected empty string when no filename")
	}
}

func TestSourceContextReturnsEmptyWhenFileNotFound(t *testing.T) {
	filename := "/nonexistent/path/file.ghoul"
	pos := &SourcePosition{Ln: 1, Col: 1, Filename: &filename}
	if pos.SourceContext() != "" {
		t.Error("expected empty string when file not found")
	}
}

func TestSourceContextSimple(t *testing.T) {
	lines := []string{
		"(define x 10)",
		"(define y 20)",
		"(+ x z)",
		"(+ x y)",
	}
	ctx := SourceContextFromLines(lines, 3, 2)
	if !strings.Contains(ctx, "(+ x z)") {
		t.Errorf("expected error line in context, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "^") {
		t.Errorf("expected caret in context, got:\n%s", ctx)
	}
}

func TestSourceContextShowsEnclosingExpression(t *testing.T) {
	lines := []string{
		"(define (fizzbuzz n)",
		"  (cond",
		"    ((eq? 0 (mod n 15)) \"FizzBuzz\")",
		"    ((eq? 0 (mod n 3)) (fizz))",
		"    ((eq? 0 (mod n 5)) \"Buzz\")",
		"    (else n)))",
	}
	// Error on line 4 col 28 — inside (fizz) which is inside the cond, inside the define
	ctx := SourceContextFromLines(lines, 4, 28)
	// Should include lines from the enclosing define
	if !strings.Contains(ctx, "(define (fizzbuzz n)") {
		t.Errorf("expected enclosing define in context, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "(fizz)") {
		t.Errorf("expected error line in context, got:\n%s", ctx)
	}
}

func TestSourceContextShowsTwoLinesAfter(t *testing.T) {
	lines := []string{
		"(define x 10)",
		"(foo x)",
		"(define y 20)",
		"(define z 30)",
		"(define w 40)",
	}
	ctx := SourceContextFromLines(lines, 2, 2)
	if !strings.Contains(ctx, "(define y 20)") {
		t.Errorf("expected 1 line after in context, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "(define z 30)") {
		t.Errorf("expected 2 lines after in context, got:\n%s", ctx)
	}
}

func TestSourceContextNearEndOfFile(t *testing.T) {
	lines := []string{
		"(define x 10)",
		"(foo x)",
	}
	ctx := SourceContextFromLines(lines, 2, 2)
	if !strings.Contains(ctx, "(foo x)") {
		t.Errorf("expected error line in context, got:\n%s", ctx)
	}
}

func TestSourceContextNestedLambda(t *testing.T) {
	lines := []string{
		"(define outer",
		"  (lambda (x)",
		"    (define inner",
		"      (lambda (y)",
		"        (+ x unknown)))",
		"    (inner 5)))",
	}
	// Error on line 5, inside the inner lambda
	ctx := SourceContextFromLines(lines, 5, 12)
	// Should walk back to at least the inner lambda definition
	if !strings.Contains(ctx, "(define inner") {
		t.Errorf("expected enclosing define in context, got:\n%s", ctx)
	}
}

func TestSourceContextTopLevelError(t *testing.T) {
	lines := []string{
		"(foo 1)",
	}
	ctx := SourceContextFromLines(lines, 1, 2)
	if !strings.Contains(ctx, "(foo 1)") {
		t.Errorf("expected error line, got:\n%s", ctx)
	}
}

func TestNodeTypeNameAllTypes(t *testing.T) {
	cases := []struct {
		node     *Node
		expected string
	}{
		{BoolNode(true), "boolean"},
		{IntNode(42), "integer"},
		{FloatNode(3.14), "float"},
		{StrNode("hi"), "string"},
		{IdentNode("foo"), "identifier"},
		{ScopedIdentNode("x", map[uint64]bool{1: true}), "identifier"},
		{QuoteNodeVal(IntNode(1)), "quoted expression"},
		{NewListNode([]*Node{IntNode(1)}), "list"},
		{Nil, "empty list"},
		{ForeignNodeVal(42), "foreign value"},
		{MummyNodeVal(42, "int"), "mummy value"},
	}
	for _, c := range cases {
		result := NodeTypeName(c.node)
		if result != c.expected {
			t.Errorf("NodeTypeName(%s) = %q, expected %q", c.node.Repr(), result, c.expected)
		}
	}
}

func TestMacroExpansionLocationColumn(t *testing.T) {
	callSite := &SourcePosition{Ln: 5, Col: 12}
	loc := &MacroExpansionLocation{MacroName: "my-mac", CallSite: callSite}
	if loc.Column() != 12 {
		t.Errorf("expected column 12, got %d", loc.Column())
	}
}

func TestMacroExpansionLocationSourceContext(t *testing.T) {
	tmpFile := t.TempDir() + "/test.ghoul"
	os.WriteFile(tmpFile, []byte("(define x 1)\n(my-mac x)\n(define y 2)\n"), 0644)

	callSite := &SourcePosition{Ln: 2, Col: 2, Filename: &tmpFile}
	loc := &MacroExpansionLocation{MacroName: "my-mac", CallSite: callSite}
	ctx := loc.SourceContext()
	if !strings.Contains(ctx, "(my-mac x)") {
		t.Errorf("expected macro call in context, got:\n%s", ctx)
	}
}

func TestMacroExpansionLocationSourceContextNoFile(t *testing.T) {
	callSite := &SourcePosition{Ln: 1, Col: 1}
	loc := &MacroExpansionLocation{MacroName: "mac", CallSite: callSite}
	if loc.SourceContext() != "" {
		t.Error("expected empty context when no filename")
	}
}

func TestSourceContextEmptyLines(t *testing.T) {
	lines := []string{}
	ctx := SourceContextFromLines(lines, 1, 1)
	if ctx != "" {
		t.Error("expected empty context for empty lines")
	}
}

func TestSourceContextOutOfBoundsLine(t *testing.T) {
	lines := []string{"(foo)"}
	ctx := SourceContextFromLines(lines, 5, 1)
	if ctx != "" {
		t.Error("expected empty context for out-of-bounds line")
	}
}

func TestSourcePositionSourceContextEmptyErrorLine(t *testing.T) {
	tmpFile := t.TempDir() + "/test.ghoul"
	os.WriteFile(tmpFile, []byte("(foo)\n\n(bar)\n"), 0644)

	pos := &SourcePosition{Ln: 2, Col: 1, Filename: &tmpFile}
	if pos.SourceContext() != "" {
		t.Error("expected empty context when error line is blank")
	}
}

func TestSourcePositionSourceContextLineBeyondFile(t *testing.T) {
	tmpFile := t.TempDir() + "/test.ghoul"
	os.WriteFile(tmpFile, []byte("(foo)\n"), 0644)

	pos := &SourcePosition{Ln: 99, Col: 1, Filename: &tmpFile}
	if pos.SourceContext() != "" {
		t.Error("expected empty context when line is beyond file length")
	}
}

func TestSourceContextWithNonexistentFilename(t *testing.T) {
	bogus := "/no/such/file/ever.ghoul"
	pos := &SourcePosition{Ln: 1, Col: 1, Filename: &bogus}
	if pos.SourceContext() != "" {
		t.Error("expected empty context for nonexistent file")
	}
	// String() should still include the filename even if unreadable
	if !strings.Contains(pos.String(), "ever.ghoul") {
		t.Errorf("expected filename in String(), got '%s'", pos.String())
	}
}

func TestSourceContextWithDirectoryAsFilename(t *testing.T) {
	dir := t.TempDir()
	pos := &SourcePosition{Ln: 1, Col: 1, Filename: &dir}
	if pos.SourceContext() != "" {
		t.Error("expected empty context when filename is a directory")
	}
}

func TestSourceContextWithEmptyFilename(t *testing.T) {
	empty := ""
	pos := &SourcePosition{Ln: 1, Col: 1, Filename: &empty}
	if pos.SourceContext() != "" {
		t.Error("expected empty context for empty filename string")
	}
}

func TestSourceContextWithZeroLine(t *testing.T) {
	tmpFile := t.TempDir() + "/test.ghoul"
	os.WriteFile(tmpFile, []byte("(foo)\n"), 0644)

	pos := &SourcePosition{Ln: 0, Col: 1, Filename: &tmpFile}
	if pos.SourceContext() != "" {
		t.Error("expected empty context for line 0")
	}
}

func TestSourceContextWithNegativeLine(t *testing.T) {
	tmpFile := t.TempDir() + "/test.ghoul"
	os.WriteFile(tmpFile, []byte("(foo)\n"), 0644)

	pos := &SourcePosition{Ln: -1, Col: 1, Filename: &tmpFile}
	if pos.SourceContext() != "" {
		t.Error("expected empty context for negative line")
	}
}

func TestSourceContextWithUnreadableFile(t *testing.T) {
	// Create a file, parse it, then delete it before the error is formatted
	tmpFile := t.TempDir() + "/disappearing.ghoul"
	os.WriteFile(tmpFile, []byte("(foo)\n"), 0644)

	pos := &SourcePosition{Ln: 1, Col: 2, Filename: &tmpFile}
	// Verify it works while file exists
	ctx := pos.SourceContext()
	if !strings.Contains(ctx, "(foo)") {
		t.Fatalf("expected context while file exists, got: %s", ctx)
	}

	// Delete the file
	os.Remove(tmpFile)
	ctx = pos.SourceContext()
	if ctx != "" {
		t.Error("expected empty context after file deletion")
	}
}

func TestSourcePositionImplementsCodeLocation(t *testing.T) {
	var loc CodeLocation = &SourcePosition{Ln: 1, Col: 1}
	if loc.Line() != 1 {
		t.Error("should implement CodeLocation")
	}
}

func TestMacroExpansionLocationString(t *testing.T) {
	callSite := &SourcePosition{Ln: 5, Col: 1}
	loc := &MacroExpansionLocation{MacroName: "my-swap", CallSite: callSite}

	if loc.Line() != 5 {
		t.Errorf("expected line from call site, got %d", loc.Line())
	}
	if loc.String() != "5:1 in expansion of 'my-swap'" {
		t.Errorf("expected '5:1 in expansion of 'my-swap'', got '%s'", loc.String())
	}
}

func TestMacroExpansionLocationImplementsCodeLocation(t *testing.T) {
	var loc CodeLocation = &MacroExpansionLocation{
		MacroName: "test",
		CallSite:  &SourcePosition{Ln: 1, Col: 1},
	}
	if loc.Line() != 1 {
		t.Error("should implement CodeLocation")
	}
}
