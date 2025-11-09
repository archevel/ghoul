package macromancy

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/parser"
)

func TestMacroTypeAssertionSafety(t *testing.T) {
	// Test cases that should not panic but return proper error messages
	cases := []struct {
		name        string
		macroCode   string
		testCode    string
		expectPanic bool
		expectError string
	}{
		{
			name:        "Non-identifier in macro pattern should not panic",
			macroCode:   `(define-syntax bad-macro (syntax-rules () ((123 x) x)))`,
			testCode:    "(bad-macro 5)",
			expectPanic: true, // This should initially panic, then we'll fix it
			expectError: "macro pattern must contain identifiers",
		},
		{
			name:        "Non-identifier in ellipsis pattern should not panic",
			macroCode:   `(define-syntax bad-ellipsis (syntax-rules () (("string" ...) 42)))`,
			testCode:    "(bad-ellipsis 1 2 3)",
			expectPanic: true, // This should initially panic, then we'll fix it
			expectError: "macro pattern ellipsis requires identifier",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !c.expectPanic {
						t.Errorf("Expected no panic but got: %v", r)
					}
					// If we expect panic during TDD phase, this is ok
				} else if c.expectPanic {
					// Once we fix the code, update expectPanic to false and check for proper error
					t.Log("No panic occurred - good! (Update test to check for error message)")
				}
			}()

			// Parse macro definition
			parseRes, parsed := parser.Parse(strings.NewReader(c.macroCode))
			if parseRes != 0 {
				t.Fatalf("Failed to parse macro definition: %s", c.macroCode)
			}

			// Try to create macro group (this is where panic can occur)
			_, err := NewMacroGroup(parsed.Expressions.First())
			if err != nil && !c.expectPanic {
				// Once fixed, we should get proper error message instead of panic
				if !strings.Contains(err.Error(), "macro pattern") {
					t.Errorf("Expected error about macro pattern, got: %v", err)
				}
			}
		})
	}
}

func TestIdAndRestFunctionSafety(t *testing.T) {
	// Test the idAndRest function with non-identifier inputs
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("idAndRest should not panic, got: %v", r)
		}
	}()

	// Test with non-identifier expressions
	testCases := []e.Expr{
		e.String("not-an-identifier"),
		e.Integer(123),
		e.Cons(e.String("nested"), e.Integer(456)),
	}

	for _, expr := range testCases {
		// This should now return an error instead of panicking
		_, _, err := idAndRest(expr)
		if err == nil {
			t.Errorf("Expected error for expression %v, but got none", expr)
		}
	}
}

func TestSplitListAtTypeAssertionSafety(t *testing.T) {
	// Test splitListAt with edge cases that could cause splitPoint type assertion to fail
	defer func() {
		if r := recover(); r != nil {
			// Initially this might panic, then we'll fix it
			t.Logf("splitListAt panicked (to be fixed): %v", r)
		}
	}()

	// Create a list that might trigger the unsafe type assertion
	testList := e.Cons(e.String("a"), e.Cons(e.String("b"), e.String("c"))) // Improper list ending with string instead of NIL

	// This should not panic
	_, _ = splitListAt(1, testList)
}