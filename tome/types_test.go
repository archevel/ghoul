package tome

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestNumberPredicate(t *testing.T) {
	r1, _ := evalWithStdlib("(number? 42)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("integer is a number") }
	r2, _ := evalWithStdlib("(number? 3.14)")
	if !r2.Equiv(e.BoolNode(true)) { t.Error("float is a number") }
	r3, _ := evalWithStdlib(`(number? "hi")`)
	if !r3.Equiv(e.BoolNode(false)) { t.Error("string is not a number") }
}

func TestIntegerPredicate(t *testing.T) {
	r1, _ := evalWithStdlib("(integer? 42)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("42 is an integer") }
	r2, _ := evalWithStdlib("(integer? 3.14)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("3.14 is not an integer") }
}

func TestFloatPredicate(t *testing.T) {
	r1, _ := evalWithStdlib("(float? 3.14)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("3.14 is a float") }
	r2, _ := evalWithStdlib("(float? 42)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("42 is not a float") }
}

func TestStringPredicate(t *testing.T) {
	r1, _ := evalWithStdlib(`(string? "hello")`)
	if !r1.Equiv(e.BoolNode(true)) { t.Error("string is a string") }
	r2, _ := evalWithStdlib("(string? 42)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("integer is not a string") }
}

func TestBooleanPredicate(t *testing.T) {
	r1, _ := evalWithStdlib("(boolean? #t)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("#t is a boolean") }
	r2, _ := evalWithStdlib("(boolean? 42)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("integer is not a boolean") }
}

func TestListPredicate(t *testing.T) {
	r1, _ := evalWithStdlib("(list? (list 1 2 3))")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("list is a list") }
	r2, _ := evalWithStdlib("(list? 42)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("integer is not a list") }
	r3, _ := evalWithStdlib("(list? (list))")
	if !r3.Equiv(e.BoolNode(true)) { t.Error("empty list is a list") }
}

func TestIntegerToFloat(t *testing.T) {
	result, _ := evalWithStdlib("(integer->float 42)")
	if !result.Equiv(e.FloatNode(42.0)) { t.Errorf("got %s", result.Repr()) }
}

func TestFloatToInteger(t *testing.T) {
	result, _ := evalWithStdlib("(float->integer 3.7)")
	if !result.Equiv(e.IntNode(3)) { t.Errorf("expected 3 (truncated), got %s", result.Repr()) }
}

func TestListPredicateImproperList(t *testing.T) {
	// Cons pair that isn't a proper list
	result, err := evalWithStdlib("(list? (cons 1 2))")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.BoolNode(false)) { t.Error("improper list should not be list?") }
}

func TestIntegerToFloatTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(integer->float "hi")`)
	if err == nil { t.Fatal("expected type error") }
}

func TestFloatToIntegerTypeError(t *testing.T) {
	_, err := evalWithStdlib("(float->integer 42)")
	if err == nil { t.Fatal("expected type error") }
}
