package tome

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestPrintln(t *testing.T) {
	// Just verify it doesn't error — output goes to stdout
	result, err := evalWithStdlib(`(println "hello")`)
	if err != nil { t.Fatal(err) }
	if result != e.NIL { t.Errorf("expected NIL, got %s", result.Repr()) }
}

func TestPrintlnNonString(t *testing.T) {
	result, err := evalWithStdlib(`(println 42)`)
	if err != nil { t.Fatal(err) }
	if result != e.NIL { t.Errorf("expected NIL, got %s", result.Repr()) }
}

func TestPrint(t *testing.T) {
	result, err := evalWithStdlib(`(print "hello")`)
	if err != nil { t.Fatal(err) }
	if result != e.NIL { t.Errorf("expected NIL, got %s", result.Repr()) }
}

func TestPrintNonString(t *testing.T) {
	result, err := evalWithStdlib(`(print 42)`)
	if err != nil { t.Fatal(err) }
	if result != e.NIL { t.Errorf("expected NIL, got %s", result.Repr()) }
}
