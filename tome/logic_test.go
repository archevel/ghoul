package tome

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestNot(t *testing.T) {
	r1, _ := evalWithStdlib("(not #t)")
	if !r1.Equiv(e.BoolNode(false)) { t.Error("(not #t) should be #f") }
	r2, _ := evalWithStdlib("(not #f)")
	if !r2.Equiv(e.BoolNode(true)) { t.Error("(not #f) should be #t") }
}

func TestNotTypeError(t *testing.T) {
	_, err := evalWithStdlib("(not 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestAnd(t *testing.T) {
	r1, _ := evalWithStdlib("(and #t #t)")
	if !r1.Equiv(e.BoolNode(true)) { t.Error("(and #t #t) should be #t") }
	r2, _ := evalWithStdlib("(and #t #f)")
	if !r2.Equiv(e.BoolNode(false)) { t.Error("(and #t #f) should be #f") }
}

func TestAndSecondArgTypeError(t *testing.T) {
	_, err := evalWithStdlib("(and #t 42)")
	if err == nil { t.Fatal("expected type error") }
}

func TestAndTypeError(t *testing.T) {
	_, err := evalWithStdlib("(and 1 #t)")
	if err == nil { t.Fatal("expected type error") }
}
