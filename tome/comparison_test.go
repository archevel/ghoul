package tome

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestLessThanIntegers(t *testing.T) {
	result, _ := evalWithStdlib("(< 1 2)")
	if !result.Equiv(e.BoolNode(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestLessThanFalse(t *testing.T) {
	result, _ := evalWithStdlib("(< 2 1)")
	if !result.Equiv(e.BoolNode(false)) { t.Errorf("expected #f, got %s", result.Repr()) }
}

func TestLessThanFloats(t *testing.T) {
	result, _ := evalWithStdlib("(< 1.0 2.0)")
	if !result.Equiv(e.BoolNode(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestLessThanMixed(t *testing.T) {
	result, _ := evalWithStdlib("(< 1 2.0)")
	if !result.Equiv(e.BoolNode(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestGreaterThan(t *testing.T) {
	result, _ := evalWithStdlib("(> 5 3)")
	if !result.Equiv(e.BoolNode(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestGreaterThanFalse(t *testing.T) {
	result, _ := evalWithStdlib("(> 3 5)")
	if !result.Equiv(e.BoolNode(false)) { t.Errorf("expected #f, got %s", result.Repr()) }
}

func TestLessEqual(t *testing.T) {
	r1, _ := evalWithStdlib("(<= 3 3)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("3 <= 3 should be #t") }
	r2, _ := evalWithStdlib("(<= 2 3)")
	if !r2.Equiv(e.BoolNode(true)) { t.Error("2 <= 3 should be #t") }
	r3, _ := evalWithStdlib("(<= 4 3)")
	if !r3.Equiv(e.BoolNode(false)) { t.Error("4 <= 3 should be #f") }
}

func TestGreaterEqual(t *testing.T) {
	r1, _ := evalWithStdlib("(>= 3 3)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("3 >= 3 should be #t") }
	r2, _ := evalWithStdlib("(>= 4 3)")
	if !r2.Equiv(e.BoolNode(true)) { t.Error("4 >= 3 should be #t") }
	r3, _ := evalWithStdlib("(>= 2 3)")
	if !r3.Equiv(e.BoolNode(false)) { t.Error("2 >= 3 should be #f") }
}

func TestNumericEquality(t *testing.T) {
	r1, _ := evalWithStdlib("(= 5 5)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("5 = 5 should be #t") }
	r2, _ := evalWithStdlib("(= 5 6)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("5 = 6 should be #f") }
}

func TestNumericEqualityFloats(t *testing.T) {
	r, _ := evalWithStdlib("(= 5.0 5.0)")
	if !r.Equiv(e.BoolNode(true)) { t.Error("5.0 = 5.0 should be #t") }
	r2, _ := evalWithStdlib("(= 5.0 6.0)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("5.0 = 6.0 should be #f") }
}

func TestNumericEqualityMixed(t *testing.T) {
	r1, _ := evalWithStdlib("(= 5 5.0)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("5 = 5.0 should be #t") }
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
	if !r.Equiv(e.BoolNode(true)) { t.Error("3.0 <= 3.0 should be #t") }
}

func TestGreaterEqualFloat(t *testing.T) {
	r, _ := evalWithStdlib("(>= 3.0 3.0)")
	if !r.Equiv(e.BoolNode(true)) { t.Error("3.0 >= 3.0 should be #t") }
}

func TestGreaterThanFloats(t *testing.T) {
	r, _ := evalWithStdlib("(> 5.0 3.0)")
	if !r.Equiv(e.BoolNode(true)) { t.Error("5.0 > 3.0 should be #t") }
}

func TestLessEqualMixed(t *testing.T) {
	r, _ := evalWithStdlib("(<= 3 3.0)")
	if !r.Equiv(e.BoolNode(true)) { t.Error("3 <= 3.0 should be #t") }
}

func TestGreaterEqualMixed(t *testing.T) {
	r, _ := evalWithStdlib("(>= 3.0 3)")
	if !r.Equiv(e.BoolNode(true)) { t.Error("3.0 >= 3 should be #t") }
}

// --- eq? structural equality ---

func TestEqStrings(t *testing.T) {
	r1, _ := evalWithStdlib(`(eq? "hello" "hello")`)
	if !r1.Equiv(e.BoolNode(true)) { t.Error(`"hello" eq? "hello" should be #t`) }
	r2, _ := evalWithStdlib(`(eq? "hello" "world")`)
	if !r2.Equiv(e.BoolNode(false)) { t.Error(`"hello" eq? "world" should be #f`) }
}

func TestEqIntegers(t *testing.T) {
	r, _ := evalWithStdlib("(eq? 42 42)")
	if !r.Equiv(e.BoolNode(true)) { t.Error("42 eq? 42 should be #t") }
}

func TestEqComputedIntegers(t *testing.T) {
	r, _ := evalWithStdlib("(eq? 6 (+ 1 2 3))")
	if !r.Equiv(e.BoolNode(true)) { t.Error("6 eq? (+ 1 2 3) should be #t") }
}

func TestEqLists(t *testing.T) {
	r, _ := evalWithStdlib("(eq? (list 1 2 3) (list 1 2 3))")
	if !r.Equiv(e.BoolNode(true)) { t.Error("equal lists should be eq?") }
}

func TestEqNestedLists(t *testing.T) {
	r, _ := evalWithStdlib("(eq? (list 1 (list 2 3)) (list 1 (list 2 3)))")
	if !r.Equiv(e.BoolNode(true)) { t.Error("equal nested lists should be eq?") }
}

func TestEqUnequalLists(t *testing.T) {
	r, _ := evalWithStdlib("(eq? (list 1 2) (list 1 3))")
	if !r.Equiv(e.BoolNode(false)) { t.Error("unequal lists should not be eq?") }
}

func TestEqDifferentLengthLists(t *testing.T) {
	r, _ := evalWithStdlib("(eq? (list 1 2) (list 1 2 3))")
	if !r.Equiv(e.BoolNode(false)) { t.Error("different length lists should not be eq?") }
}

func TestEqEmptyLists(t *testing.T) {
	r, _ := evalWithStdlib("(eq? (list) (list))")
	if !r.Equiv(e.BoolNode(true)) { t.Error("empty lists should be eq?") }
}

func TestEqBooleans(t *testing.T) {
	r1, _ := evalWithStdlib("(eq? #t #t)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("#t eq? #t should be #t") }
	r2, _ := evalWithStdlib("(eq? #t #f)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("#t eq? #f should be #f") }
}

func TestEqMixedTypes(t *testing.T) {
	r1, _ := evalWithStdlib(`(eq? 1 "1")`)
	if !r1.Equiv(e.BoolNode(false)) { t.Error(`1 eq? "1" should be #f`) }
	r2, _ := evalWithStdlib("(eq? 1 #t)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("1 eq? #t should be #f") }
	r3, _ := evalWithStdlib("(eq? (list 1) 1)")
	if !r3.Equiv(e.BoolNode(false)) { t.Error("(list 1) eq? 1 should be #f") }
}

func TestEqNil(t *testing.T) {
	r, _ := evalWithStdlib("(eq? (list) '())")
	if !r.Equiv(e.BoolNode(true)) { t.Error("empty list eq? '() should be #t") }
}

func TestEqIntFloat(t *testing.T) {
	r, _ := evalWithStdlib("(eq? 5 5.0)")
	if !r.Equiv(e.BoolNode(true)) { t.Error("5 eq? 5.0 should be #t (numeric coercion)") }
}

// --- = numeric equality errors on non-numbers ---

func TestNumericEqRejectsStrings(t *testing.T) {
	_, err := evalWithStdlib(`(= "a" "a")`)
	if err == nil { t.Fatal("= should reject strings") }
}

func TestNumericEqRejectsBooleans(t *testing.T) {
	_, err := evalWithStdlib("(= #t #t)")
	if err == nil { t.Fatal("= should reject booleans") }
}
