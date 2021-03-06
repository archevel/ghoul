package ghoul

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

const guidingScript = `
(define fiz-buz (lambda (n)
  (cond ((and (eq? 0 (mod n 3)) (eq? 0 (mod n 5))) "FizzBuzz")
        ((eq? 0 (mod n 3)) "Fizz")
        ((eq? 0 (mod n 5)) "Buzz")
        (else n))))

(define loop (lambda (i m body)
  (cond ((< i m)
    (begin (body i) (loop (+ 1 i) m body))))))


(define do-fizz-buzz-to (lambda (n) 
  (loop 0 (+ 1 n) 
    (lambda (i) 
      (println (fiz-buz i))))))

(do-fizz-buzz-to 10)
`

func ExampleGuiding() {
	r := strings.NewReader(guidingScript)
	g := New()

	g.Process(r)
	// Output:
	// FizzBuzz
	// 1
	// 2
	// Fizz
	// 4
	// Buzz
	// Fizz
	// 7
	// 8
	// Fizz
	// Buzz

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
		t.Error("Got nil for error when processing '%s'", in)
	}
}

func TestExpandsMacrosBeforeEvaluating(t *testing.T) {
	g := New()
	in := "(define-syntax baz (syntax-rules () (((baz x y) (+ x y))))) (baz 1 2)"
	expected := e.Integer(3)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got an error '%s' when processing '%s'", err, in)
	}

	if !expected.Equiv(res) {
		t.Errorf("Got %s, expected %s when evaluating '%s'", res.Repr(), expected.Repr(), in)
	}
}

func TestBasicBuiltInFunctions(t *testing.T) {
	cases := []struct {
		in  string
		out e.Expr
	}{
		{"(eq? 1 1)", e.Boolean(true)},
		{"(eq? 1 \"3\")", e.Boolean(false)},

		{"(and #t #t)", e.Boolean(true)},
		{"(and #t #f)", e.Boolean(false)},

		{"(< 1 2)", e.Boolean(true)},
		{"(< 2 1)", e.Boolean(false)},

		{"(mod 9 3)", e.Integer(0)},
		{"(mod 1 3)", e.Integer(1)},

		{"(+ 9 3)", e.Integer(12)},
		{"(+ 1 3)", e.Integer(4)},
		{"(+ -1 3)", e.Integer(2)},
	}

	for _, c := range cases {
		g := New()
		res, err := g.Process(strings.NewReader(c.in))

		if err != nil {
			t.Errorf("'%s' yielded an unexpected error: %s", c.in, err.Error())
		}

		if !c.out.Equiv(res) {
			t.Errorf("'%s' failed, expected %s, got %s", c.in, c.out.Repr(), res.Repr())
		}
	}
}

func ExampleTestPrintln() {
	g := New()
	g.Process(strings.NewReader(`(println "hello, world")`))

	// Output:
	// hello, world
}
