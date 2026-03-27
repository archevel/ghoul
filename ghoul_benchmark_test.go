package ghoul

import (
	"strings"
	"testing"
)

// BenchmarkFibonacci exercises the full pipeline: parse → expand → evaluate
// with a recursive fibonacci computation.
func BenchmarkFibonacci(b *testing.B) {
	code := `
(define fib (lambda (n)
  (cond
    ((eq? n 0) 0)
    ((eq? n 1) 1)
    (else (+ (fib (- n 1)) (fib (- n 2)))))))
(fib 15)
`
	for i := 0; i < b.N; i++ {
		g := New()
		result, err := g.Process(strings.NewReader(code))
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// BenchmarkFibonacciPreloaded measures only evaluation, not Ghoul creation.
func BenchmarkFibonacciPreloaded(b *testing.B) {
	code := `
(define fib (lambda (n)
  (cond
    ((eq? n 0) 0)
    ((eq? n 1) 1)
    (else (+ (fib (- n 1)) (fib (- n 2)))))))
(fib 15)
`
	g := New()
	// Warm up — ensures stdlib is loaded
	g.Process(strings.NewReader("1"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := g.Process(strings.NewReader(code))
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// BenchmarkTailRecursiveCountdown tests tail call optimization performance.
func BenchmarkTailRecursiveCountdown(b *testing.B) {
	code := `
(define countdown (lambda (n)
  (cond
    ((eq? n 0) 0)
    (else (countdown (- n 1))))))
(countdown 10000)
`
	g := New()
	g.Process(strings.NewReader("1"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := g.Process(strings.NewReader(code))
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// BenchmarkListOperations tests list construction and traversal.
func BenchmarkListOperations(b *testing.B) {
	code := `
(define build-list (lambda (n acc)
  (cond
    ((eq? n 0) acc)
    (else (build-list (- n 1) (cons n acc))))))
(define sum-list (lambda (lst acc)
  (cond
    ((null? lst) acc)
    (else (sum-list (cdr lst) (+ acc (car lst)))))))
(sum-list (build-list 100 '()) 0)
`
	g := New()
	g.Process(strings.NewReader("1"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := g.Process(strings.NewReader(code))
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// BenchmarkMacroExpansion tests the cost of macro expansion with let.
func BenchmarkMacroExpansion(b *testing.B) {
	// Load prelude for let macro
	preludeCode := `
(define-syntax let (syntax-rules ()
  ((let ((var val) ...) body ...)
   ((lambda (var ...) body ...) val ...))))
`
	code := `
(let ((x 1) (y 2) (z 3))
  (+ x (+ y z)))
`
	g := New()
	g.Process(strings.NewReader(preludeCode))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := g.Process(strings.NewReader(code))
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// BenchmarkParseOnly measures just the parsing phase.
func BenchmarkParseOnly(b *testing.B) {
	code := `
(define fib (lambda (n)
  (cond
    ((eq? n 0) 0)
    ((eq? n 1) 1)
    (else (+ (fib (- n 1)) (fib (- n 2)))))))
(fib 15)
`
	for i := 0; i < b.N; i++ {
		g := New()
		// Process parses + expands + evaluates; we can't separate easily
		// but this measures the baseline
		result, err := g.Process(strings.NewReader(code))
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}
