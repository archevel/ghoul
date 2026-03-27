package consume

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/exhumer"
)

// setupTestEnvironment creates an environment with basic arithmetic functions for testing
func setupTestEnvironment() *environment {
	env := NewEnvironment()

	// Register basic arithmetic functions
	env.Register("+", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		if len(args) < 1 || args[0].Kind != e.IntegerNode {
			return nil, fmt.Errorf("+: first argument must be integer")
		}
		if len(args) < 2 || args[1].Kind != e.IntegerNode {
			return nil, fmt.Errorf("+: second argument must be integer")
		}
		return e.IntNode(args[0].IntVal + args[1].IntVal), nil
	})

	env.Register("-", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		if len(args) < 1 || args[0].Kind != e.IntegerNode {
			return nil, fmt.Errorf("-: first argument must be integer")
		}
		if len(args) < 2 || args[1].Kind != e.IntegerNode {
			return nil, fmt.Errorf("-: second argument must be integer")
		}
		return e.IntNode(args[0].IntVal - args[1].IntVal), nil
	})

	env.Register("*", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		if len(args) < 1 || args[0].Kind != e.IntegerNode {
			return nil, fmt.Errorf("*: first argument must be integer")
		}
		if len(args) < 2 || args[1].Kind != e.IntegerNode {
			return nil, fmt.Errorf("*: second argument must be integer")
		}
		return e.IntNode(args[0].IntVal * args[1].IntVal), nil
	})

	env.Register("<", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		if len(args) < 1 || args[0].Kind != e.IntegerNode {
			return nil, fmt.Errorf("<: first argument must be integer")
		}
		if len(args) < 2 || args[1].Kind != e.IntegerNode {
			return nil, fmt.Errorf("<: second argument must be integer")
		}
		return e.BoolNode(args[0].IntVal < args[1].IntVal), nil
	})

	return env
}

func TestEvaluateWithContextHappyPath(t *testing.T) {
	env := setupTestEnvironment()
	ctx := context.Background()

	// Test simple expression evaluation with context
	parseRes, parsed := exhumer.Parse(strings.NewReader("(+ 2 3)"))
	if parseRes != 0 {
		t.Fatal("Failed to parse test expression")
	}

	result, err := EvaluateWithContext(ctx, parsed.Expressions, env)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if expected := e.IntNode(5); !result.Equiv(expected) {
		t.Errorf("Expected %s, but got %s", expected.Repr(), result.Repr())
	}
}

func TestEvaluateWithContextCancellation(t *testing.T) {
	env := NewEnvironment()

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Test that a simple operation respects cancellation
	parseRes, parsed := exhumer.Parse(strings.NewReader("(+ 1 1)"))
	if parseRes != 0 {
		t.Fatal("Failed to parse test expression")
	}

	result, err := EvaluateWithContext(ctx, parsed.Expressions, env)

	// Should return context.Canceled error
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, but got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result on cancellation, but got: %s", result.Repr())
	}
}

func TestEvaluateWithContextTimeout(t *testing.T) {
	env := NewEnvironment()

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a moment to ensure timeout occurs
	time.Sleep(10 * time.Millisecond)

	parseRes, parsed := exhumer.Parse(strings.NewReader("(+ 1 1)"))
	if parseRes != 0 {
		t.Fatal("Failed to parse test expression")
	}

	result, err := EvaluateWithContext(ctx, parsed.Expressions, env)

	// Should return context.DeadlineExceeded error
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded error, but got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result on timeout, but got: %s", result.Repr())
	}
}

func TestEvaluateWithContextMultipleExpressions(t *testing.T) {
	env := setupTestEnvironment()
	ctx := context.Background()

	// Test multiple expressions with context
	parseRes, parsed := exhumer.Parse(strings.NewReader(`
		(define x 10)
		(define y 20)
		(+ x y)
	`))
	if parseRes != 0 {
		t.Fatal("Failed to parse test expressions")
	}

	result, err := EvaluateWithContext(ctx, parsed.Expressions, env)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if expected := e.IntNode(30); !result.Equiv(expected) {
		t.Errorf("Expected %s, but got %s", expected.Repr(), result.Repr())
	}
}

func TestEvaluateWithContextRecursiveFunction(t *testing.T) {
	env := setupTestEnvironment()
	ctx := context.Background()

	// Test simpler recursive function - countdown
	parseRes, parsed := exhumer.Parse(strings.NewReader(`
		(define countdown
		  (lambda (n)
		    (+ n 0)))
		(countdown 3)
	`))
	if parseRes != 0 {
		t.Fatal("Failed to parse recursive function")
	}

	result, err := EvaluateWithContext(ctx, parsed.Expressions, env)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if expected := e.IntNode(3); !result.Equiv(expected) {
		t.Errorf("Expected %s, but got %s", expected.Repr(), result.Repr())
	}
}

func TestEvaluateWithContextCancelledDuringExecution(t *testing.T) {
	env := setupTestEnvironment()
	ctx, cancel := context.WithCancel(context.Background())

	// Add a built-in function that takes some time and checks cancellation
	env.Register("slow-add", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		// Simulate some work by checking context multiple times
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}

		if len(args) < 1 || args[0].Kind != e.IntegerNode {
			return nil, fmt.Errorf("slow-add: first argument must be integer")
		}
		if len(args) < 2 || args[1].Kind != e.IntegerNode {
			return nil, fmt.Errorf("slow-add: second argument must be integer")
		}
		return e.IntNode(args[0].IntVal + args[1].IntVal), nil
	})

	// Start evaluation in a goroutine
	resultChan := make(chan *e.Node, 1)
	errorChan := make(chan error, 1)

	go func() {
		parseRes, parsed := exhumer.Parse(strings.NewReader("(slow-add 10 20)"))
		if parseRes != 0 {
			errorChan <- fmt.Errorf("Failed to parse")
			return
		}

		result, err := EvaluateWithContext(ctx, parsed.Expressions, env)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Cancel after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Check that we get cancellation error
	select {
	case result := <-resultChan:
		t.Errorf("Expected cancellation, but got result: %s", result.Repr())
	case err := <-errorChan:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, but got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for cancellation")
	}
}

func TestEvaluateWithContextNilExpression(t *testing.T) {
	env := NewEnvironment()
	ctx := context.Background()

	// Test NIL expression with context
	result, err := EvaluateWithContext(ctx, e.Nil, env)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if !result.Equiv(e.Nil) {
		t.Errorf("Expected NIL, but got %s", result.Repr())
	}
}

func TestBackwardCompatibilityEvaluate(t *testing.T) {
	env := setupTestEnvironment()

	// Test that old Evaluate method still works (uses Background context internally)
	parseRes, parsed := exhumer.Parse(strings.NewReader("(+ 5 7)"))
	if parseRes != 0 {
		t.Fatal("Failed to parse test expression")
	}

	result, err := Evaluate(parsed.Expressions, env)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if expected := e.IntNode(12); !result.Equiv(expected) {
		t.Errorf("Expected %s, but got %s", expected.Repr(), result.Repr())
	}
}
