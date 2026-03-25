package stdlib

import (
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

func TestLessThanIntegers(t *testing.T) {
	result, _ := evalWithStdlib("(< 1 2)")
	if !result.Equiv(e.Boolean(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestLessThanFalse(t *testing.T) {
	result, _ := evalWithStdlib("(< 2 1)")
	if !result.Equiv(e.Boolean(false)) { t.Errorf("expected #f, got %s", result.Repr()) }
}

func TestLessThanFloats(t *testing.T) {
	result, _ := evalWithStdlib("(< 1.0 2.0)")
	if !result.Equiv(e.Boolean(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestLessThanMixed(t *testing.T) {
	result, _ := evalWithStdlib("(< 1 2.0)")
	if !result.Equiv(e.Boolean(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestGreaterThan(t *testing.T) {
	result, _ := evalWithStdlib("(> 5 3)")
	if !result.Equiv(e.Boolean(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestGreaterThanFalse(t *testing.T) {
	result, _ := evalWithStdlib("(> 3 5)")
	if !result.Equiv(e.Boolean(false)) { t.Errorf("expected #f, got %s", result.Repr()) }
}

func TestLessEqual(t *testing.T) {
	r1, _ := evalWithStdlib("(<= 3 3)")
	if !r1.Equiv(e.Boolean(true)) { t.Error("3 <= 3 should be #t") }
	r2, _ := evalWithStdlib("(<= 2 3)")
	if !r2.Equiv(e.Boolean(true)) { t.Error("2 <= 3 should be #t") }
	r3, _ := evalWithStdlib("(<= 4 3)")
	if !r3.Equiv(e.Boolean(false)) { t.Error("4 <= 3 should be #f") }
}

func TestGreaterEqual(t *testing.T) {
	r1, _ := evalWithStdlib("(>= 3 3)")
	if !r1.Equiv(e.Boolean(true)) { t.Error("3 >= 3 should be #t") }
	r2, _ := evalWithStdlib("(>= 4 3)")
	if !r2.Equiv(e.Boolean(true)) { t.Error("4 >= 3 should be #t") }
	r3, _ := evalWithStdlib("(>= 2 3)")
	if !r3.Equiv(e.Boolean(false)) { t.Error("2 >= 3 should be #f") }
}

func TestNumericEquality(t *testing.T) {
	r1, _ := evalWithStdlib("(= 5 5)")
	if !r1.Equiv(e.Boolean(true)) { t.Error("5 = 5 should be #t") }
	r2, _ := evalWithStdlib("(= 5 6)")
	if !r2.Equiv(e.Boolean(false)) { t.Error("5 = 6 should be #f") }
}

func TestNumericEqualityMixed(t *testing.T) {
	r1, _ := evalWithStdlib("(= 5 5.0)")
	if !r1.Equiv(e.Boolean(true)) { t.Error("5 = 5.0 should be #t") }
}

func TestComparisonTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(< "a" 1)`)
	if err == nil { t.Fatal("expected type error") }
}

func TestComparisonSecondArgError(t *testing.T) {
	_, err := evalWithStdlib(`(> 1 "b")`)
	if err == nil { t.Fatal("expected type error") }
}

func TestNumericEqualityTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(= "a" "b")`)
	if err == nil { t.Fatal("expected type error") }
}

func TestLessEqualFloat(t *testing.T) {
	r, _ := evalWithStdlib("(<= 3.0 3.0)")
	if !r.Equiv(e.Boolean(true)) { t.Error("3.0 <= 3.0 should be #t") }
}

func TestGreaterEqualFloat(t *testing.T) {
	r, _ := evalWithStdlib("(>= 3.0 3.0)")
	if !r.Equiv(e.Boolean(true)) { t.Error("3.0 >= 3.0 should be #t") }
}

func TestEqStructural(t *testing.T) {
	r1, _ := evalWithStdlib(`(eq? "hello" "hello")`)
	if !r1.Equiv(e.Boolean(true)) { t.Error("expected #t") }
	r2, _ := evalWithStdlib(`(eq? "hello" "world")`)
	if !r2.Equiv(e.Boolean(false)) { t.Error("expected #f") }
}
