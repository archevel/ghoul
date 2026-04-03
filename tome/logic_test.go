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
