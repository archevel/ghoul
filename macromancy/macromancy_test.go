package macromancy

import (
	"strings"
	"testing"

	"github.com/archevel/ghoul/parser"
)

func TestMacromancerDoesNotModifyNonMacroCode(t *testing.T) {
	code := `(a b c)`
	expected := `(a b c)`

	runMacroTest(t, code, expected)
}

func TestMacromancerRemovesValidMacrosDefinitionsItEncounters(t *testing.T) {
	code := `(define-syntax foo (syntax-rules () (((foo x) (x biz)))))`
	expected := ``

	runMacroTest(t, code, expected)
}

func TestMacromancerAppliesFoundMacrosDefinitions(t *testing.T) {
	code := `(define-syntax foo (syntax-rules () (((foo x) (x biz)))))  (foo bar)`
	expected := `(bar biz)`

	runMacroTest(t, code, expected)
}

func TestMacromancerDoesNotAlterCodeBeforeEncounteredMacros(t *testing.T) {
	code := `(foo bar) (define-syntax foo (syntax-rules () (((foo x) (x biz)))))`
	expected := `(foo bar)`

	runMacroTest(t, code, expected)
}

func TestMacromancerOnlyReadsMacrosOnAllLevels(t *testing.T) {
	code := `(baz (define-syntax foo (syntax-rules () (((foo x) (x biz)))))) (foo bar)`
	expected := `(baz) (bar biz)`

	runMacroTest(t, code, expected)
}

func TestMacromancerOnlyExpandsListsStartingWithMatchIds(t *testing.T) {
	code := `(define-syntax foo (syntax-rules () (((foo x) (x biz))))) (fiz foo bar)`
	expected := `(fiz foo bar)`

	runMacroTest(t, code, expected)
}

func TestMacromancerCanExpandMacrosInSubLists(t *testing.T) {
	code := `(define-syntax foo (syntax-rules () (((foo x) (x biz))))) ((fiz buz) (foo  bar))`
	expected := `((fiz buz) (bar biz))`

	runMacroTest(t, code, expected)
}

func TestMacromancerCanHandlePairs(t *testing.T) {
	code := `(define-syntax foo (syntax-rules () (((foo x) (x biz))))) ((fiz . buz) (foo  bar))`
	expected := `((fiz . buz) (bar biz))`

	runMacroTest(t, code, expected)
}

func TestMacromancerCanExpandToNewMacro(t *testing.T) {
	code := `
(define-syntax foo
  (syntax-rules ()
	  (
			((foo x) ((define-syntax x (syntax-rules () (((x) (biz))))) x)))))
(foo bar) (bar)
`
	expected := `(biz) (biz)`

	runMacroTest(t, code, expected)
}

func runMacroTest(t *testing.T, code string, expected string) {
	ok, parsedCode := parser.Parse(strings.NewReader(code))
	if ok != 0 {
		t.Errorf("Failed to parse code: %s\n", code)
	}

	var transformer Transformer = NewMacromancer()

	ok, parsedExpected := parser.Parse(strings.NewReader(expected))
	if ok != 0 {
		t.Errorf("Failed to parse expected: %s\n", expected)
	}

	mancedCode, err := transformer.Transform(parsedCode.Expressions)

	if err != nil {
		t.Errorf(`Got error: "%s" when mancing code %s`, err, code)
	}

	if err == nil && !mancedCode.Equiv(parsedExpected.Expressions) {
		t.Errorf("Expected code:\n'%s'\n\n to yield:\n '%s'\n\n after macromancy transform, but got: \n'%s'", parsedCode.Expressions.Repr(), parsedExpected.Expressions.Repr(), mancedCode.Repr())
	}
}
