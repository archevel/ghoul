package ghoul

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	e "github.com/archevel/ghoul/expressions"
)

func TestProcessWithContextHappyPath(t *testing.T) {
	ghoul := New()
	ctx := context.Background()

	// Test simple expression processing with context
	reader := strings.NewReader("(+ 3 4)")

	result, err := ghoul.ProcessWithContext(ctx, reader)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if expected := e.Integer(7); !result.Equiv(expected) {
		t.Errorf("Expected %s, but got %s", expected.Repr(), result.Repr())
	}
}

func TestProcessWithContextCancellation(t *testing.T) {
	ghoul := New()

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	reader := strings.NewReader("(+ 1 2)")

	result, err := ghoul.ProcessWithContext(ctx, reader)

	// Should return chained context.Canceled error
	if err == nil {
		t.Error("Expected error, but got nil")
	} else if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error (possibly chained), but got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result on cancellation, but got: %s", result.Repr())
	}
}

func TestProcessWithContextTimeout(t *testing.T) {
	ghoul := New()

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a moment to ensure timeout occurs
	time.Sleep(10 * time.Millisecond)

	reader := strings.NewReader("(+ 10 20)")

	result, err := ghoul.ProcessWithContext(ctx, reader)

	// Should return context.DeadlineExceeded error
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error (possibly chained), but got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result on timeout, but got: %s", result.Repr())
	}
}

func TestProcessWithContextComplexProgram(t *testing.T) {
	ghoul := New()
	ctx := context.Background()

	// Test complex program with context using only available functions
	program := `
		(define double (lambda (x) (+ x x)))
		(define sum-of-doubles (lambda (a b) (+ (double a) (double b))))
		(sum-of-doubles 3 4)
	`
	reader := strings.NewReader(program)

	result, err := ghoul.ProcessWithContext(ctx, reader)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	// double(3) + double(4) = 6 + 8 = 14
	if expected := e.Integer(14); !result.Equiv(expected) {
		t.Errorf("Expected %s, but got %s", expected.Repr(), result.Repr())
	}
}

func TestProcessBackwardCompatibility(t *testing.T) {
	ghoul := New()

	// Test that old Process method still works (uses Background context internally)
	reader := strings.NewReader("(+ 8 9)")

	result, err := ghoul.Process(reader)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if expected := e.Integer(17); !result.Equiv(expected) {
		t.Errorf("Expected %s, but got %s", expected.Repr(), result.Repr())
	}
}

func TestProcessWithContextParseError(t *testing.T) {
	ghoul := New()
	ctx := context.Background()

	// Test that parse errors are still returned properly with context
	reader := strings.NewReader("(+ 1 2 3") // Missing closing paren

	result, err := ghoul.ProcessWithContext(ctx, reader)

	// Should get parse error, not context error
	if err == nil {
		t.Error("Expected parse error, but got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result on parse error, but got: %s", result.Repr())
	}

	// Error should mention parsing
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error message, but got: %v", err)
	}
}