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

func guidingExample() {
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
		t.Errorf("Got nil for error when processing '%s'", in)
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

func TestHygienicMacroOrWithTmp(t *testing.T) {
	g := New()
	in := `
(define-syntax my-or (syntax-rules () (((my-or a b) (begin (define tmp a) (cond (tmp tmp) (else b)))))))
(define tmp 5)
(my-or #f tmp)
`
	expected := e.Integer(5)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestHygienicMacroSwapWithTmp(t *testing.T) {
	g := New()
	in := `
(define-syntax my-swap (syntax-rules () (((my-swap x y) (begin (define tmp x) (set! x y) (set! y tmp))))))
(define a 10)
(define b 20)
(my-swap a b)
a
`
	expected := e.Integer(20)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestHygienicMacroSwapWithTmpVariable(t *testing.T) {
	// User has a variable named "tmp" — the macro's tmp should not capture it
	g := New()
	in := `
(define-syntax my-swap (syntax-rules () (((my-swap x y) (begin (define tmp x) (set! x y) (set! y tmp))))))
(define tmp 10)
(define other 20)
(my-swap tmp other)
tmp
`
	expected := e.Integer(20)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
		t.Errorf("Expected %s (user's tmp should be swapped), got %s", expected.Repr(), res.Repr())
	}
}

func TestGeneralTransformerAlways42(t *testing.T) {
	g := New()
	in := `
(define-syntax always-42
  (lambda (stx) 42))
(always-42 anything)
`
	expected := e.Integer(42)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
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
	expected := e.Integer(10)
	res, err := g.Process(strings.NewReader(in))

	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestNestedMacroExpansion(t *testing.T) {
	// One macro expands to code that uses another macro
	g := New()
	in := `
(define-syntax add-one (syntax-rules () (((add-one x) (+ x 1)))))
(define-syntax add-two (syntax-rules () (((add-two x) (add-one (add-one x))))))
(add-two 3)
`
	expected := e.Integer(5)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestNestedMacrosWithSameTmpVariable(t *testing.T) {
	// Both macros introduce "tmp" — they should get distinct marks
	g := New()
	in := `
(define-syntax save-first (syntax-rules () (((save-first a b) (begin (define tmp a) tmp)))))
(define-syntax save-second (syntax-rules () (((save-second a b) (begin (define tmp b) (save-first tmp a))))))
(save-second 10 20)
`
	// save-second expands to: (begin (define tmp$1 20) (save-first tmp$1 10))
	// save-first then expands to: (begin (define tmp$2 tmp$1) tmp$2)
	// tmp$2 = tmp$1 = 20, so result is 20
	expected := e.Integer(20)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestNestedMacrosWithUserTmpVariable(t *testing.T) {
	// User has "tmp", outer macro introduces "tmp", inner macro introduces "tmp"
	// All three should be distinct
	g := New()
	in := `
(define-syntax wrap-a (syntax-rules () (((wrap-a x) (begin (define tmp x) tmp)))))
(define-syntax wrap-b (syntax-rules () (((wrap-b x) (begin (define tmp x) (wrap-a tmp))))))
(define tmp 99)
(wrap-b tmp)
`
	// wrap-b expands to: (begin (define tmp$1 tmp) (wrap-a tmp$1))
	// wrap-a expands to: (begin (define tmp$2 tmp$1) tmp$2)
	// tmp$2 = tmp$1 = user's tmp = 99
	expected := e.Integer(99)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func TestMacroExpandingToDefineSyntax(t *testing.T) {
	// A macro that defines another macro (meta-macro)
	g := New()
	in := `
(define-syntax def-adder (syntax-rules () (((def-adder name val) (define-syntax name (syntax-rules () (((name x) (+ x val)))))))))
(def-adder add-five 5)
(add-five 10)
`
	expected := e.Integer(15)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
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
	expected := e.Integer(42)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
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
	expected := e.Integer(42)
	res, err := g.Process(strings.NewReader(in))
	if err != nil {
		t.Errorf("Got error: %s", err)
	}
	if !expected.Equiv(res) {
		t.Errorf("Expected %s, got %s", expected.Repr(), res.Repr())
	}
}

func testPrintlnExample() {
	g := New()
	g.Process(strings.NewReader(`(println "hello, world")`))

	// Output:
	// hello, world
}
