package evaluator

import (
	"strings"
	"testing"

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
		{`(define "x" 1)`, "Bad syntax: no valid identifier given"},
		{`(define . x)`, "undefined identifier: define"},
		{`(define . (x . 1))`, "Bad syntax in procedure application"},
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
		{"(begin . `foo`)", "undefined identifier: begin"},
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
		{"(begin . `foo`)", "undefined identifier: begin"},
		{"(define x . `foo`)", "Bad syntax in procedure application"},
		{"(lambda () . `foo`)", "Bad syntax in procedure application"},
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
	expectedErrorMessage := "assignment disallowed"
	testInputResultsInError(in, expectedErrorMessage, t)
}

func TestAssignmentNeedsToConformToFormat(t *testing.T) {

	cases := []struct {
		in                   string
		expectedErrorMessage string
	}{

		{`(set! x)`, "undefined identifier: x"},

		{`(define x 1) (set! x)`, "undefined identifier: set!"},
		{`(set! x . 1)`, "Bad syntax in procedure application"},
		{`(set! x 1 2)`, "undefined identifier: x"},
	}
	for i, c := range cases {
		t.Logf("Test #%d", i)
		testInputResultsInError(c.in, c.expectedErrorMessage, t)
	}
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

/*
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

			if _, found := parsed.PositionOf(*ee.ErrorList.(*e.Pair)); !found {
				t.Error("Expected error to have a recorded position, but nothing found")
			}
		default:
			t.Errorf("Did not get an EvaluationError: %T(%+v)", err, err)
		}

	}
}
/**/
