package ghoul

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/bones"
)

// --- Prelude auto-loading tests ---

func TestPreludeAutoLoaded(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(let ((x 1) (y 2)) (+ x y))"))
	if err != nil {
		t.Fatalf("let should work with auto-loaded prelude: %s", err)
	}
	if !res.Equiv(e.IntNode(3)) {
		t.Errorf("expected 3, got %s", res.Repr())
	}
}

func TestPreludeLetStar(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(let* ((x 1) (y (+ x 1))) y)"))
	if err != nil {
		t.Fatalf("let* should work with auto-loaded prelude: %s", err)
	}
	if !res.Equiv(e.IntNode(2)) {
		t.Errorf("expected 2, got %s", res.Repr())
	}
}

func TestPreludeWhen(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(when #t 42)"))
	if err != nil {
		t.Fatalf("when should work with auto-loaded prelude: %s", err)
	}
	if !res.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", res.Repr())
	}
}

func TestPreludeAnd(t *testing.T) {
	cases := []struct{ code, desc string; expected *e.Node }{
		{"(and)", "(and) should be #t", e.BoolNode(true)},
		{"(and #t)", "(and #t) should be #t", e.BoolNode(true)},
		{"(and #t #t)", "(and #t #t) should be #t", e.BoolNode(true)},
		{"(and #t #f)", "(and #t #f) should be #f", e.BoolNode(false)},
		{"(and #t #t #t)", "(and #t #t #t) should be #t", e.BoolNode(true)},
		{"(and #t #f #t)", "(and #t #f #t) should be #f", e.BoolNode(false)},
		{"(and 42)", "(and 42) should be 42", e.IntNode(42)},
	}
	for _, tc := range cases {
		g := New()
		res, err := g.Process(strings.NewReader(tc.code))
		if err != nil { t.Fatalf("%s: %v", tc.desc, err) }
		if !res.Equiv(tc.expected) { t.Errorf("%s, got %s", tc.desc, res.Repr()) }
	}
}

func TestPreludeAndShortCircuit(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(and #f (/ 1 0))"))
	if err != nil { t.Fatalf("expected short-circuit, got error: %v", err) }
	if !res.Equiv(e.BoolNode(false)) { t.Error("(and #f (/ 1 0)) should be #f") }
}

func TestPreludeOr(t *testing.T) {
	cases := []struct{ code, desc string; expected *e.Node }{
		{"(or)", "(or) should be #f", e.BoolNode(false)},
		{"(or #f)", "(or #f) should be #f", e.BoolNode(false)},
		{"(or #t #t)", "(or #t #t) should be #t", e.BoolNode(true)},
		{"(or #t #f)", "(or #t #f) should be #t", e.BoolNode(true)},
		{"(or #f #t)", "(or #f #t) should be #t", e.BoolNode(true)},
		{"(or #f #f)", "(or #f #f) should be #f", e.BoolNode(false)},
		{"(or #f #f #t)", "(or #f #f #t) should be #t", e.BoolNode(true)},
		{"(or #f #f #f)", "(or #f #f #f) should be #f", e.BoolNode(false)},
		{"(or #f 42)", "(or #f 42) should be 42", e.IntNode(42)},
	}
	for _, tc := range cases {
		g := New()
		res, err := g.Process(strings.NewReader(tc.code))
		if err != nil { t.Fatalf("%s: %v", tc.desc, err) }
		if !res.Equiv(tc.expected) { t.Errorf("%s, got %s", tc.desc, res.Repr()) }
	}
}

func TestPreludeOrShortCircuit(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(or #t (/ 1 0))"))
	if err != nil { t.Fatalf("expected short-circuit, got error: %v", err) }
	if !res.Equiv(e.BoolNode(true)) { t.Error("(or #t (/ 1 0)) should be #t") }
}

func TestBareSkipsPrelude(t *testing.T) {
	g := NewBare()
	_, err := g.Process(strings.NewReader("(let ((x 1)) x)"))
	if err == nil {
		t.Fatal("expected error — let should not be available without prelude")
	}
}

func TestFailsOnUnparsableCode(t *testing.T) {
	g := New()

	_, err := g.Process(strings.NewReader(")"))

	if err == nil {
		t.Error("Got nil for error when parsing ')'")
	}
}

func TestYieldsEvaluationErrorWhenThereIsAnErrror(t *testing.T) {
	g := New()
	in := "(baz 1 2 3)"
	_, err := g.Process(strings.NewReader(in))

	if err == nil {
		t.Errorf("Got nil for error when processing '%s'", in)
	}
}

func TestExpandsMacrosBeforeEvaluating(t *testing.T) {
	g := New()
	in := "(define-syntax baz (syntax-rules () ((baz x y) (+ x y)))) (baz 1 2)"
	expected := e.IntNode(3)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got an error '%s' when processing '%s'", err, in)
	}

	if !res.Equiv(expected) {
		t.Errorf("Got %s, expected %s when evaluating '%s'", res.Repr(), expected.Repr(), in)
	}
}

func TestBasicBuiltInFunctions(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{"(eq? 1 1)", e.BoolNode(true)},
		{"(eq? 1 \"3\")", e.BoolNode(false)},

		{"(and #t #t)", e.BoolNode(true)},
		{"(and #t #f)", e.BoolNode(false)},

		{"(< 1 2)", e.BoolNode(true)},
		{"(< 2 1)", e.BoolNode(false)},

		{"(mod 9 3)", e.IntNode(0)},
		{"(mod 1 3)", e.IntNode(1)},

		{"(+ 9 3)", e.IntNode(12)},
		{"(+ 1 3)", e.IntNode(4)},
		{"(+ -1 3)", e.IntNode(2)},
	}

	for _, c := range cases {
		g := New()
		res, err := g.Process(strings.NewReader(c.in))

		if err != nil {
			t.Errorf("'%s' yielded an unexpected error: %s", c.in, err.Error())
		}

		if !res.Equiv(c.out) {
			t.Errorf("'%s' failed, expected %s, got %s", c.in, c.out.Repr(), res.Repr())
		}
	}
}

func TestHygienicMacroOrWithTmp(t *testing.T) {
	g := New()
	in := `
(define-syntax my-or (syntax-rules () ((my-or a b) (begin (define tmp a) (cond (tmp tmp) (else b))))))
(define tmp 5)
(my-or #f tmp)
`
	expected := e.IntNode(5)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestHygienicMacroSwapWithTmp(t *testing.T) {
	g := New()
	in := `
(define-syntax my-swap (syntax-rules () ((my-swap x y) (begin (define tmp x) (set! x y) (set! y tmp)))))
(define a 10)
(define b 20)
(my-swap a b)
a
`
	expected := e.IntNode(20)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestHygienicMacroSwapWithTmpVariable(t *testing.T) {
	// User has a variable named "tmp" — the macro's tmp should not capture it
	g := New()
	in := `
(define-syntax my-swap (syntax-rules () ((my-swap x y) (begin (define tmp x) (set! x y) (set! y tmp)))))
(define tmp 10)
(define other 20)
(my-swap tmp other)
tmp
`
	expected := e.IntNode(20)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s (user's tmp should be swapped), got %s", expected.Repr(), res.Repr())
	}
}

func TestSyntaxRulesWithEllipsis(t *testing.T) {
	g := New()
	// my-list collects all arguments into a list structure via begin+define
	// This tests that ... in pattern captures variable args
	// and ... in template splices them back
	in := `
(define-syntax my-begin (syntax-rules () ((my-begin x ...) (begin x ...))))
(my-begin (define a 1) (define b 2) (+ a b))
`
	expected := e.IntNode(3)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if res != nil && !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestSyntaxRulesWithEllipsisMultipleArgs(t *testing.T) {
	g := New()
	in := `
(define-syntax add-all (syntax-rules () ((add-all x y ...) (+ x (+ y ...)))))
(add-all 1 2 3)
`
	// (add-all 1 2 3) should expand to (+ 1 (+ 2 3))
	// But this requires ... to work in both pattern matching and template expansion
	expected := e.IntNode(6)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if res != nil && !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestSyntaxRulesLiterals(t *testing.T) {
	g := New()
	// "else" is a literal — it must match exactly, not bind as a pattern variable
	in := `
(define-syntax my-if (syntax-rules (else)
  ((my-if test then else alt) (cond (test then) (else alt)))))
(my-if #t 1 else 2)
`
	expected := e.IntNode(1)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if res != nil && !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestSyntaxRulesLiteralPreventsBinding(t *testing.T) {
	g := New()
	// Without literals, "arrow" in the pattern would be a pattern variable
	// that captures anything. With "arrow" as a literal, the pattern only
	// matches when "arrow" appears literally at that position.
	in := `
(define-syntax test-lit (syntax-rules (arrow)
  ((test-lit x arrow y) (+ x y))))
(test-lit 3 arrow 4)
`
	expected := e.IntNode(7)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if res != nil && !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestSyntaxRulesLiteralRejectsNonLiteral(t *testing.T) {
	g := New()
	// When a non-literal identifier is in the position where a literal is expected,
	// the pattern should NOT match. Currently without literal support,
	// "arrow" would match anything as a pattern variable.
	in := `
(define-syntax test-lit (syntax-rules (arrow)
  ((test-lit x arrow y) (+ x y))))
(test-lit 3 blah 4)
`
	// Should fail because "blah" doesn't match literal "arrow"
	_, err := g.Process(strings.NewReader(in))
	if err == nil {
		t.Error("Expected error because 'blah' should not match literal 'arrow'")
	}
}

func TestSyntaxRulesLiteralNotBoundAsVariable(t *testing.T) {
	g := New()
	// "else" is a literal, so in the body it should appear as itself (not substituted)
	// and in the pattern it should not capture the input as a variable
	in := `
(define-syntax my-if (syntax-rules (else)
  ((my-if test then else alt) (cond (test then) (else alt)))))
(my-if (eq? 1 2) 10 else 20)
`
	expected := e.IntNode(20)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if res != nil && !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestGeneralTransformerAlways42(t *testing.T) {
	g := New()
	in := `
(define-syntax always-42
  (lambda (stx) 42))
(always-42 anything)
`
	expected := e.IntNode(42)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestGeneralTransformerWithListConstruction(t *testing.T) {
	g := New()
	// Transformer that takes (add-3 x) and produces (+ x 3)
	in := `
(define-syntax add-3
  (lambda (stx)
    (define arg (car (cdr stx)))
    (list '+ (syntax->datum arg) 3)))
(add-3 7)
`
	expected := e.IntNode(10)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestNestedMacroExpansion(t *testing.T) {
	// One macro expands to code that uses another macro
	g := New()
	in := `
(define-syntax add-one (syntax-rules () ((add-one x) (+ x 1))))
(define-syntax add-two (syntax-rules () ((add-two x) (add-one (add-one x)))))
(add-two 3)
`
	expected := e.IntNode(5)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestNestedMacrosWithSameTmpVariable(t *testing.T) {
	// Both macros introduce "tmp" — they should get distinct marks
	g := New()
	in := `
(define-syntax save-first (syntax-rules () ((save-first a b) (begin (define tmp a) tmp))))
(define-syntax save-second (syntax-rules () ((save-second a b) (begin (define tmp b) (save-first tmp a)))))
(save-second 10 20)
`
	// save-second expands to: (begin (define tmp$1 20) (save-first tmp$1 10))
	// save-first then expands to: (begin (define tmp$2 tmp$1) tmp$2)
	// tmp$2 = tmp$1 = 20, so result is 20
	expected := e.IntNode(20)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestNestedMacrosWithUserTmpVariable(t *testing.T) {
	// User has "tmp", outer macro introduces "tmp", inner macro introduces "tmp"
	// All three should be distinct
	g := New()
	in := `
(define-syntax wrap-a (syntax-rules () ((wrap-a x) (begin (define tmp x) tmp))))
(define-syntax wrap-b (syntax-rules () ((wrap-b x) (begin (define tmp x) (wrap-a tmp)))))
(define tmp 99)
(wrap-b tmp)
`
	// wrap-b expands to: (begin (define tmp$1 tmp) (wrap-a tmp$1))
	// wrap-a expands to: (begin (define tmp$2 tmp$1) tmp$2)
	// tmp$2 = tmp$1 = user's tmp = 99
	expected := e.IntNode(99)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestMacroExpandingToDefineSyntax(t *testing.T) {
	// A macro that defines another macro (meta-macro)
	g := New()
	in := `
(define-syntax def-adder (syntax-rules () ((def-adder name val) (define-syntax name (syntax-rules () ((name x) (+ x val)))))))
(def-adder add-five 5)
(add-five 10)
`
	expected := e.IntNode(15)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestGeneralTransformerHygienePassthroughIdentifiers(t *testing.T) {
	// A general transformer that passes input identifiers through.
	// Those identifiers must NOT get marked — only transformer-introduced ones should.
	g := New()
	in := `
(define-syntax passthrough
  (lambda (stx)
    (car (cdr stx))))
(define x 42)
(passthrough x)
`
	expected := e.IntNode(42)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestGeneralTransformerHygieneIntroducedBinding(t *testing.T) {
	// Transformer introduces a binding named "x".
	// User also has "x". They should not conflict.
	g := New()
	in := `
(define-syntax bind-x-to-99
  (lambda (stx)
    (list 'begin (list 'define 'x 99) (car (cdr stx)))))
(define x 42)
(bind-x-to-99 x)
`
	// The transformer produces: (begin (define x 99) x)
	// With correct hygiene: the 'x in (define x 99) is introduced by the transformer
	// and gets marked, while the x from user input passes through unmarked.
	// So user's x (42) is returned, not 99.
	expected := e.IntNode(42)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestSyntaxRulesMultipleClauses(t *testing.T) {
	// Multiple clauses are tried in order; the first matching pattern wins.
	macro := `(define-syntax my-op (syntax-rules ()
  ((my-op x y) (+ x y))
  ((my-op x) (+ x 1))))`

	cases := []struct {
		name     string
		call     string
		expected *e.Node
	}{
		{"second clause matches single arg", "(my-op 10)", e.IntNode(11)},
		{"first clause matches two args", "(my-op 3 4)", e.IntNode(7)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := New()
			res, err := g.Process(strings.NewReader(macro + "\n" + c.call))
			if err != nil {
				t.Fatalf("Got error: %s", err)
			}
			if res != nil && !res.Equiv(c.expected) {
				t.Errorf("Expected %s, got %s", c.expected.Repr(), res.Repr())
			}
		})
	}
}

func TestQuotedIdentifiersInTransformerGetHygiene(t *testing.T) {
	g := New()
	in := `
(define-syntax bind-and-return
  (lambda (stx)
    (list 'begin
          (list 'define 'x 99)
          'x)))
(define x 42)
(bind-and-return)
`
	// The transformer introduces 'x via quote. With correct hygiene,
	// the macro's x should not shadow the user's x.
	// bind-and-return defines its own scoped x=99 and returns that,
	// so result is 99 (the macro's own x, not the user's).
	// The user's x=42 remains untouched.
	expected := e.IntNode(99)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if res != nil && !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestErrorShowsSourceContext(t *testing.T) {
	tmpFile := t.TempDir() + "/test.ghoul"
	os.WriteFile(tmpFile, []byte("(define x 10)\n(define y 20)\n(+ x z)\n(+ x y)\n"), 0644)

	g := New()
	_, err := g.ProcessFile(tmpFile)
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "(+ x z)") {
		t.Errorf("expected error to show source line '(+ x z)', got:\n%s", errMsg)
	}
	if !strings.Contains(errMsg, "^") {
		t.Errorf("expected error to show caret pointer, got:\n%s", errMsg)
	}
}

func TestErrorShowsSourceContextForMacro(t *testing.T) {
	tmpFile := t.TempDir() + "/test_macro.ghoul"
	os.WriteFile(tmpFile, []byte("(define-syntax bad (syntax-rules () ((bad x) (+ x missing))))\n(define a 5)\n(bad a)\n"), 0644)

	g := New()
	_, err := g.ProcessFile(tmpFile)
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "(bad a)") {
		t.Errorf("expected error to show macro call site '(bad a)', got:\n%s", errMsg)
	}
}

func TestErrorWithoutFilenameShowsNoSourceContext(t *testing.T) {
	g := New()
	_, err := g.Process(strings.NewReader("(foo 1)"))
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "^") {
		t.Errorf("expected no source context for non-file input, got:\n%s", errMsg)
	}
}

func TestErrorShowsFilenameInLocation(t *testing.T) {
	tmpFile := t.TempDir() + "/myfile.ghoul"
	os.WriteFile(tmpFile, []byte("(foo 1)\n"), 0644)

	g := New()
	_, err := g.ProcessFile(tmpFile)
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "myfile.ghoul") {
		t.Errorf("expected filename in error, got:\n%s", errMsg)
	}
}

func TestNestedEllipsisLetMacro(t *testing.T) {
	g := New()
	in := `
(define-syntax let (syntax-rules ()
  ((let ((var val) ...) body ...)
   ((lambda (var ...) body ...) val ...))))
(let ((x 1) (y 2)) (+ x y))
`
	expected := e.IntNode(3)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestNestedEllipsisLetMacroMultipleBody(t *testing.T) {
	g := New()
	in := `
(define-syntax let (syntax-rules ()
  ((let ((var val) ...) body ...)
   ((lambda (var ...) body ...) val ...))))
(let ((x 10) (y 20)) (define z (+ x y)) z)
`
	expected := e.IntNode(30)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestNestedEllipsisLetMacroEmptyBindings(t *testing.T) {
	g := New()
	in := `
(define-syntax let (syntax-rules ()
  ((let ((var val) ...) body ...)
   ((lambda (var ...) body ...) val ...))))
(let () 42)
`
	expected := e.IntNode(42)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestWildcardInSyntaxRules(t *testing.T) {
	g := New()
	in := `
(define-syntax first-of (syntax-rules ()
  ((first-of x _) x)))
(first-of 42 99)
`
	expected := e.IntNode(42)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(expected) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestPreludeLetMacro(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(let ((x 10) (y 20)) (+ x y))"))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(30)) {
		t.Errorf("Expected 30, got %s", res.Repr())
	}
}

func TestPreludeLetStarMacro(t *testing.T) {
	g := New()
	// let* allows each binding to reference the previous ones
	res, err := g.Process(strings.NewReader("(let* ((x 1) (y (+ x 1)) (z (+ y 1))) z)"))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(3)) {
		t.Errorf("Expected 3, got %s", res.Repr())
	}
}

func TestPreludeLetStarEmptyBindings(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(let* () 42)"))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(42)) {
		t.Errorf("Expected 42, got %s", res.Repr())
	}
}

func TestPreludeLetStarSingleBinding(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(let* ((x 10)) (+ x 5))"))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(15)) {
		t.Errorf("Expected 15, got %s", res.Repr())
	}
}

func TestPreludeWhenMacro(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader("(when #t 42)"))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(42)) {
		t.Errorf("Expected 42, got %s", res.Repr())
	}
}

func TestMacroExpansionErrorIncludesMacroName(t *testing.T) {
	g := New()
	_, err := g.Process(strings.NewReader(`
(define-syntax bad-mac (syntax-rules () ((bad-mac x) (+ x nonexistent))))
(bad-mac 5)
`))
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "bad-mac") {
		t.Errorf("expected error to mention macro name 'bad-mac', got: %s", errMsg)
	}
}

func TestScopedIdentifierMacroCallExpansion(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader(
		`(define-syntax add1 (syntax-rules () ((add1 x) (+ x 1)))) (add1 5)`))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(6)) {
		t.Errorf("Expected 6, got %s", res.Repr())
	}
}

func TestGeneralTransformerNestedDefineInsideLambda(t *testing.T) {
	// When a general transformer defines a function inside a lambda
	// (nested define), it should work correctly through expansion.
	g := New()
	res, err := g.Process(strings.NewReader(`
(define-syntax test-mac
  (lambda (stx)
    (define clauses (cdr (cdr stx)))
    (define clause (car clauses))
    (define pat (car clause))
    (define collect
      (lambda (p)
        (define walk
          (lambda (expr acc)
            (cond
              ((null? expr) acc)
              ((identifier? expr) (+ acc 1))
              ((pair? expr) (walk (cdr expr) (walk (car expr) acc)))
              (else acc))))
        (walk (cdr p) 0)))
    (collect pat)))
(test-mac ()
  ((test-mac x) 42))
`))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(1)) {
		t.Errorf("Expected 1, got %s", res.Repr())
	}
}

func TestPreludeSyntaxCaseAdd1(t *testing.T) {
	// Verifies that syntax-case (defined as a prelude macro) works
	// for a simple pattern-matching transformer.
	g := New()
	res, err := g.Process(strings.NewReader(`
(define-syntax add1
  (lambda (stx)
    (syntax-case stx ()
      ((add1 x) (list '+ (syntax->datum x) 1)))))
(add1 41)
`))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(42)) {
		t.Errorf("Expected 42, got %s", res.Repr())
	}
}

func TestZeroArgLambda(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader(`(define f (lambda () 42)) (f)`))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(42)) {
		t.Errorf("Expected 42, got %s", res.Repr())
	}
}

func TestZeroArgLambdaClosure(t *testing.T) {
	g := New()
	res, err := g.Process(strings.NewReader(`
(define make-counter (lambda (start)
  (define count start)
  (lambda ()
    (set! count (+ count 1))
    count)))
(define c (make-counter 0))
(c)
(c)
(c)
`))
	if err != nil {
		t.Fatalf("Got error: %s", err)
	}
	if !res.Equiv(e.IntNode(3)) {
		t.Errorf("Expected 3, got %s", res.Repr())
	}
}

// --- Module integration tests ---

func TestLoadGhoulModule(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "utils.ghl"), []byte("(define x 42) (define y 99)"), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte("(require utils) utils:x"), 0644)

	g := New()
	result, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireFromSubdirectory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0755)
	os.WriteFile(filepath.Join(dir, "lib", "helpers.ghl"), []byte("(define helper-val 123)"), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte("(require lib/helpers) lib/helpers:helper-val"), 0644)

	g := New()
	result, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(123)) {
		t.Errorf("expected 123, got %s", result.Repr())
	}
}

func TestRequireFromSubdirectoryWithAlias(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0755)
	os.WriteFile(filepath.Join(dir, "lib", "helpers.ghl"), []byte("(define x 77)"), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte("(require lib/helpers as h) h:x"), 0644)

	g := New()
	result, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(77)) {
		t.Errorf("expected 77, got %s", result.Repr())
	}
}

func TestCircularDependencyError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.ghl"), []byte("(require b) (define x 1)"), 0644)
	os.WriteFile(filepath.Join(dir, "b.ghl"), []byte("(require a) (define y 2)"), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte("(require a)"), 0644)

	g := New()
	_, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' in error, got: %v", err)
	}
}

func TestRequireGhoulModuleNameConflict(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.ghl"), []byte("(define x 1)"), 0644)
	os.WriteFile(filepath.Join(dir, "b.ghl"), []byte("(define x 2)"), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte("(require a as m) (require b as m)"), 0644)

	g := New()
	_, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err == nil {
		t.Fatal("expected name conflict error")
	}
	if !strings.Contains(err.Error(), "already defined") {
		t.Errorf("expected 'already defined' in error, got: %v", err)
	}
}

func TestRequireGhoulModuleParseError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.ghl"), []byte("(define x"), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte("(require bad)"), 0644)

	g := New()
	_, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err == nil {
		t.Fatal("expected parse error for malformed module")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected 'failed to parse' in error, got: %v", err)
	}
}

func TestRequireGhoulModuleEvalError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "broken.ghl"), []byte("(undefined-func 1 2)"), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte("(require broken)"), 0644)

	g := New()
	_, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err == nil {
		t.Fatal("expected evaluation error for broken module")
	}
}

func TestRequireMacroFromModule(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "macros.ghl"), []byte(`
(define-syntax add1 (syntax-rules () ((add1 x) (+ x 1))))
`), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte(`
(require macros as m)
(m:add1 41)
`), 0644)

	g := New()
	result, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireModuleWithFunctionAndMacro(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "utils.ghl"), []byte(`
(define double (lambda (x) (+ x x)))
(define-syntax when-positive (syntax-rules ()
  ((when-positive val body ...) (cond ((> val 0) (begin body ...))))))
`), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte(`
(require utils as u)
(u:when-positive 5 (u:double 21))
`), 0644)

	g := New()
	result, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireSameModuleFromTwoModules(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "shared.ghl"), []byte("(define val 55)"), 0644)
	os.WriteFile(filepath.Join(dir, "a.ghl"), []byte("(require shared) (define a-val shared:val)"), 0644)
	os.WriteFile(filepath.Join(dir, "b.ghl"), []byte("(require shared) (define b-val shared:val)"), 0644)
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte("(require a) (require b) a:a-val"), 0644)

	g := New()
	result, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(55)) {
		t.Errorf("expected 55, got %s", result.Repr())
	}
}

// Cross-module set! tests: verify that set! on a module-qualified binding
// does NOT propagate back to the exporting module's internal variable.
// This documents the current behavior where module exports are value copies
// in the importer's scope, not references to the exporter's scope.

func TestSetOnImportedBindingDoesNotAffectExporter(t *testing.T) {
	dir := t.TempDir()
	// Module A: defines foo and a getter that reads foo from A's own scope
	os.WriteFile(filepath.Join(dir, "a.ghl"), []byte(
		"(define foo 42) (define get-foo (lambda () foo))"), 0644)
	// Main: imports A, mutates a:foo, then calls a:get-foo to see if A noticed
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte(
		"(require a) (set! a:foo 99) (a:get-foo)"), 0644)

	g := New()
	result, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A's internal foo is still 42 — set! only changed main's local binding
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42 (A's foo unchanged), got %s", result.Repr())
	}
}

func TestSetViaExportedSetterAffectsExporter(t *testing.T) {
	dir := t.TempDir()
	// Module A: defines foo, a setter, and a getter
	os.WriteFile(filepath.Join(dir, "a.ghl"), []byte(
		"(define foo 42) (define set-foo! (lambda (v) (set! foo v))) (define get-foo (lambda () foo))"), 0644)
	// Main: imports A, calls the setter, then reads via the getter
	os.WriteFile(filepath.Join(dir, "main.ghl"), []byte(
		"(require a) (a:set-foo! 99) (a:get-foo)"), 0644)

	g := New()
	result, err := g.ProcessFile(filepath.Join(dir, "main.ghl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The setter lambda closes over A's actual foo, so the change IS visible
	if !result.Equiv(e.IntNode(99)) {
		t.Errorf("expected 99 (A's foo changed via setter), got %s", result.Repr())
	}
}
