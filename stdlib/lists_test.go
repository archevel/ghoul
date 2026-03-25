package stdlib

import (
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

func TestLength(t *testing.T) {
	result, _ := evalWithStdlib("(length (list 1 2 3))")
	if !result.Equiv(e.Integer(3)) { t.Errorf("got %s", result.Repr()) }
}

func TestLengthEmpty(t *testing.T) {
	result, _ := evalWithStdlib("(length (list))")
	if !result.Equiv(e.Integer(0)) { t.Errorf("got %s", result.Repr()) }
}

func TestAppendLists(t *testing.T) {
	result, _ := evalWithStdlib("(length (append (list 1 2) (list 3 4)))")
	if !result.Equiv(e.Integer(4)) { t.Errorf("got %s", result.Repr()) }
}

func TestAppendEmptyFirst(t *testing.T) {
	result, _ := evalWithStdlib("(length (append (list) (list 1 2)))")
	if !result.Equiv(e.Integer(2)) { t.Errorf("got %s", result.Repr()) }
}

func TestReverse(t *testing.T) {
	result, _ := evalWithStdlib("(car (reverse (list 1 2 3)))")
	if !result.Equiv(e.Integer(3)) { t.Errorf("expected 3, got %s", result.Repr()) }
}

func TestReverseEmpty(t *testing.T) {
	result, _ := evalWithStdlib("(reverse (list))")
	if result != e.NIL { t.Errorf("expected NIL, got %s", result.Repr()) }
}

func TestMap(t *testing.T) {
	result, _ := evalWithStdlib(`
(define double (lambda (x) (+ x x)))
(car (map double (list 3 4 5)))
`)
	if !result.Equiv(e.Integer(6)) { t.Errorf("expected 6, got %s", result.Repr()) }
}

func TestMapEmpty(t *testing.T) {
	result, _ := evalWithStdlib(`
(define id (lambda (x) x))
(map id (list))
`)
	if result != e.NIL { t.Errorf("expected NIL, got %s", result.Repr()) }
}

func TestFilter(t *testing.T) {
	result, _ := evalWithStdlib(`
(define even? (lambda (x) (eq? 0 (mod x 2))))
(length (filter even? (list 1 2 3 4 5 6)))
`)
	if !result.Equiv(e.Integer(3)) { t.Errorf("expected 3 even numbers, got %s", result.Repr()) }
}

func TestFilterNoneMatch(t *testing.T) {
	result, _ := evalWithStdlib(`
(define never (lambda (x) #f))
(filter never (list 1 2 3))
`)
	if result != e.NIL { t.Errorf("expected NIL, got %s", result.Repr()) }
}

func TestFoldl(t *testing.T) {
	result, _ := evalWithStdlib(`
(foldl + 0 (list 1 2 3 4))
`)
	if !result.Equiv(e.Integer(10)) { t.Errorf("expected 10, got %s", result.Repr()) }
}

func TestFoldlEmpty(t *testing.T) {
	result, _ := evalWithStdlib(`
(foldl + 0 (list))
`)
	if !result.Equiv(e.Integer(0)) { t.Errorf("expected 0, got %s", result.Repr()) }
}

func TestNullPredicate(t *testing.T) {
	r1, _ := evalWithStdlib("(null? (list))")
	if !r1.Equiv(e.Boolean(true)) { t.Error("empty list should be null") }
	r2, _ := evalWithStdlib("(null? (list 1))")
	if !r2.Equiv(e.Boolean(false)) { t.Error("non-empty list should not be null") }
}

func TestPairPredicate(t *testing.T) {
	r1, _ := evalWithStdlib("(pair? (list 1 2))")
	if !r1.Equiv(e.Boolean(true)) { t.Error("list should be a pair") }
	r2, _ := evalWithStdlib("(pair? 42)")
	if !r2.Equiv(e.Boolean(false)) { t.Error("integer should not be a pair") }
}

func TestMapCallbackError(t *testing.T) {
	_, err := evalWithStdlib(`(map (lambda (x) (/ x 0)) (list 1))`)
	if err == nil { t.Fatal("expected error from callback") }
}

func TestFilterCallbackError(t *testing.T) {
	_, err := evalWithStdlib(`(filter (lambda (x) (/ x 0)) (list 1))`)
	if err == nil { t.Fatal("expected error from callback") }
}

func TestFoldlCallbackError(t *testing.T) {
	_, err := evalWithStdlib(`(foldl (lambda (acc x) (/ x 0)) 0 (list 1))`)
	if err == nil { t.Fatal("expected error from callback") }
}

func TestCarCdr(t *testing.T) {
	r1, _ := evalWithStdlib("(car (list 1 2 3))")
	if !r1.Equiv(e.Integer(1)) { t.Errorf("got %s", r1.Repr()) }
	r2, _ := evalWithStdlib("(car (cdr (list 1 2 3)))")
	if !r2.Equiv(e.Integer(2)) { t.Errorf("got %s", r2.Repr()) }
}

func TestLengthTypeError(t *testing.T) {
	_, err := evalWithStdlib("(length 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestAppendTypeError(t *testing.T) {
	_, err := evalWithStdlib("(append 42 (list 1))")
	if err == nil { t.Fatal("expected type error") }
}

func TestReverseTypeError(t *testing.T) {
	_, err := evalWithStdlib("(reverse 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestMapListTypeError(t *testing.T) {
	_, err := evalWithStdlib("(map (lambda (x) x) 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestFilterListTypeError(t *testing.T) {
	_, err := evalWithStdlib("(filter (lambda (x) #t) 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestFoldlListTypeError(t *testing.T) {
	_, err := evalWithStdlib("(foldl + 0 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestCarError(t *testing.T) {
	_, err := evalWithStdlib("(car 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestCdrError(t *testing.T) {
	_, err := evalWithStdlib("(cdr 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestCons(t *testing.T) {
	result, _ := evalWithStdlib("(car (cons 1 (list 2 3)))")
	if !result.Equiv(e.Integer(1)) { t.Errorf("got %s", result.Repr()) }
}
