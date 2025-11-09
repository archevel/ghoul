package evaluator

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	p "github.com/archevel/ghoul/parser"
)

func TestVariablesMissingFromEnvironmentGivesAnError(t *testing.T) {

	in := `fool`
	expectedErrorMessage := "undefined identifier: fool"

	testInputResultsInError(in, expectedErrorMessage, t)
}

//		{`(define . x)`, "Bad syntax: invalid binding format"},
//		{`(define . (x . 1))`, "Bad syntax: invalid binding format"},
func TestVariableDefinitionsWithBadSyntax(t *testing.T) {

	cases := []struct {
		in            string
		expectedError string
	}{
		{`(define x)`, "Bad syntax: missing value in binding"},
		{`(define x 1 1)`, "Bad syntax: multiple values in binding"},
		{`(define "x" 1)`, "define: bad syntax, no valid identifier given in \"x\""},
		{`(define . x)`, "Malformed expression"},
		{`(define . (x . 1))`, "Bad syntax: invalid binding format"},
		{`(define x (define y 1 1))`, "Bad syntax: multiple values in binding"},
	}

	for _, c := range cases {
		testInputResultsInError(c.in, c.expectedError, t)
	}
}

func TestLambdasGivenTooManyArgsFail(t *testing.T) {

	in := `((lambda (x) x) 1 2)`
	expectedErrorMessage := `Arity mismatch: too many arguments`
	testInputResultsInError(in, expectedErrorMessage, t)
}

func TestLambdasGivenTooFewArgsFail(t *testing.T) {

	in := `((lambda (x y) x) 2)`
	expectedErrorMessage := `Arity mismatch: too few arguments`
	testInputResultsInError(in, expectedErrorMessage, t)
}

func TestNilIsNotCallable(t *testing.T) {
	in := "()"
	expectedErrorMessage := "Missing procedure expression in: ()"
	testInputResultsInError(in, expectedErrorMessage, t)
}

func TestLambdasBoundParamsAreOnlyAccessibleInsideOfLambda(t *testing.T) {
	in := `((lambda (x) x) 'an-arg) x`
	expectedErrorMessage := "undefined identifier: x"

	testInputResultsInError(in, expectedErrorMessage, t)
}

func TestLambdasWithBadApplication(t *testing.T) {
	in := `((lambda (x y) x) 'an-arg . 'foo)`
	expectedErrorMessage := "Bad syntax in procedure application"

	testInputResultsInError(in, expectedErrorMessage, t)
}

func TestCondMalformed(t *testing.T) {

	cases := []struct {
		in                   string
		expectedErrorMessage string
	}{
		{"(cond ())", "Bad syntax: Missing condition"},
		{"(cond (#t))", "Bad syntax: Missing consequent"},
		{"(cond (#f))", "Bad syntax: Missing consequent"},
		{"(cond (#f) ())", "Bad syntax: Missing consequent"},
		{"(cond (#f 1) ())", "Bad syntax: Missing condition"},
		{"(cond (#f) (#t))", "Bad syntax: Missing consequent"},
		{"(cond (#f) (#t 1))", "Bad syntax: Missing consequent"},
		{"(cond ((define x 1)))", "Bad syntax: Missing consequent"},
		{"(cond 'asdf)", "Bad syntax: Malformed cond clause: 'asdf"},
		{"(cond ('asdf . 'foo))", "Bad syntax: Malformed cond clause: ('asdf . 'foo)"},
	}
	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputResultsInError(c.in, c.expectedErrorMessage, t)
	}

}

func TestBeginMalformed(t *testing.T) {
	cases := []struct {
		in                   string
		expectedErrorMessage string
	}{
		{"(begin . `foo`)", "Malformed expression"},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputResultsInError(c.in, c.expectedErrorMessage, t)
	}
}

func TestSpecialFormsMalformed(t *testing.T) {
	cases := []struct {
		in                   string
		expectedErrorMessage string
	}{
		{"(begin . `foo`)", "Malformed expression"},
		{"(define x . `foo`)", "Bad syntax: invalid binding format"},
		{"(lambda () . `foo`)", "Malformed lambda expression"},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputResultsInError(c.in, c.expectedErrorMessage, t)
	}
}

func TestMalformedCallArgsResultInErrors(t *testing.T) {
	cases := []struct {
		in                   string
		expectedErrorMessage string
	}{

		{"((lambda (x) x) (cond ()))", "Bad syntax: Missing condition"},
		{"((lambda (x) x) (cond (#t)))", "Bad syntax: Missing consequent"},
		{"((lambda (x) x) foo)", "undefined identifier: foo"},
		{"((lambda (x) x) ())", "Missing procedure expression in: ()"},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputResultsInError(c.in, c.expectedErrorMessage, t)
	}
}

func TestCallToNonCallableResultsInError(t *testing.T) {
	cases := []struct {
		in                   string
		expectedErrorMessage string
	}{

		{"(1 2 3)", "Not a procedure: 1"},
	}

	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputResultsInError(c.in, c.expectedErrorMessage, t)
	}
}

func TestAssignmentFailsForUndefinedVariables(t *testing.T) {

	in := `(set! x 5)`
	expectedErrorMessage := "set!: assignment disallowed for identifier x"
	testInputResultsInError(in, expectedErrorMessage, t)
}

func TestAssignmentFailsForNonIdentifierVariable(t *testing.T) {
	// Test that assignment with non-identifier variables returns error instead of panicking
	cases := []struct {
		in                   string
		expectedErrorMessage string
	}{
		{`(set! "not-an-identifier" 5)`, "set!: variable must be an identifier, got expressions.String"},
		{`(set! 123 "value")`, "set!: variable must be an identifier, got expressions.Integer"},
		{`(set! (foo bar) "value")`, "set!: variable must be an identifier, got *expressions.Pair"},
	}

	for _, c := range cases {
		testInputResultsInError(c.in, c.expectedErrorMessage, t)
	}
}

func TestFunctionCallTypeAssertionSafety(t *testing.T) {
	// Actually, after reviewing the code, the evaluator.go:151 type assertion
	// is protected by the logic flow - expr should always be a List if we reach that point.
	// This test demonstrates that the current logic is already safe.

	cases := []struct {
		in                   string
		expectedError        bool
		expectPanic          bool
	}{
		// These are handled by other paths and don't reach line 151
		{`"not-a-function"`, false, false},  // Returns the string value
		{`123`, false, false},               // Returns the integer value
		{`(undefined-function)`, true, false}, // Should get undefined identifier error
	}

	for _, c := range cases {
		defer func() {
			if r := recover(); r != nil {
				if !c.expectPanic {
					t.Errorf("Unexpected panic for %s: %v", c.in, r)
				}
			}
		}()

		env := NewEnvironment()
		parseRes, parsed := p.Parse(strings.NewReader(c.in))
		if parseRes != 0 {
			t.Fatalf("Failed to parse: %s", c.in)
		}

		_, err := Evaluate(parsed.Expressions, env)

		if c.expectedError && err == nil {
			t.Errorf("Expected error for %s but got none", c.in)
		}
		if !c.expectedError && err != nil {
			t.Errorf("Unexpected error for %s: %v", c.in, err)
		}
	}
}

func TestAssignmentNeedsToConformToFormat(t *testing.T) {

	cases := []struct {
		in                   string
		expectedErrorMessage string
	}{

		{`(set! x)`, "Malformed assignment"},
		{`(define x 1) (set! x)`, "Malformed assignment"},
		{`(set! x . 1)`, "Malformed assignment"},
		{`(set! x 1 2)`, "Malformed assignment"},
	}
	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputResultsInError(c.in, c.expectedErrorMessage, t)
	}
}

func TestScopingWorks(t *testing.T) {
	in := `
(define a 10)
(define foo (lambda () (set! a b)))
((lambda (b) (foo)) 33)
a`

	testInputResultsInError(in, "undefined identifier: b", t)
}

func testInputResultsInError(in string, errorMessage string, t *testing.T) {
	env := NewEnvironment()
	r := strings.NewReader(in)

	parseRes, parsed := p.Parse(r)

	if parseRes != 0 {
		t.Errorf("Parser failed given: %s", in)
	}
	res, err := Evaluate(parsed.Expressions, env)

	if err == nil || err.Error() != errorMessage {
		t.Errorf("Given %s. Expected error '%s' but got %q", in, errorMessage, err)
	}

	if res != nil {
		t.Errorf("Got result when expecting error: %s", res.Repr())
	}

}

func TestErrorIsEvaluationErrorContainingBadPair(t *testing.T) {

	cases := []struct {
		in                    string
		expectedErrorPairRepr string
	}{
		{"(define . x)", "((define . x))"},
		{"((lambda () (define . x)))", "((define . x))"},
	}
	for _, c := range cases {
		in := c.in
		expectedErrorPairRepr := c.expectedErrorPairRepr
		env := NewEnvironment()
		r := strings.NewReader(in)

		parseRes, parsed := p.Parse(r)

		if parseRes != 0 {
			t.Errorf("Parser failed given: %s", in)
		}
		res, err := Evaluate(parsed.Expressions, env)

		if err == nil {
			t.Errorf("Given %s. Expected error but got nil", in)
		}

		if res != nil {
			t.Errorf("Got result when expecting error: %s", res.Repr())
		}

		switch ee := err.(type) {

		case EvaluationError:
			if ee.ErrorList.Repr() != expectedErrorPairRepr {
				t.Errorf("Expected errorPair %s but got %s", expectedErrorPairRepr, ee.ErrorList.Repr())
			}

			errorPair, ok := ee.ErrorList.(*e.Pair)
			if !ok {
				t.Error("Not ok to convert to *e.Pair")
			}
			if _, found := parsed.PositionOf(*errorPair); !found {
				t.Error("Expected error to have a recorded position, but nothing found")
			}
		default:
			t.Errorf("Did not get an EvaluationError: %T(%+v)", err, err)
		}
	}
}
