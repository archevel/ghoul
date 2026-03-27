package tome

import (
	"strings"
	"testing"

	ev "github.com/archevel/ghoul/consume"
	e "github.com/archevel/ghoul/bones"
	p "github.com/archevel/ghoul/exhumer"
)

func evalWithStdlib(code string) (*e.Node, error) {
	env := ev.NewEnvironment()
	RegisterAll(env)
	r := strings.NewReader(code)
	_, parsed := p.Parse(r)
	return ev.Evaluate(parsed.Expressions, env)
}

func TestAddIntegers(t *testing.T) {
	result, err := evalWithStdlib("(+ 3 4)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.IntNode(7)) { t.Errorf("expected 7, got %s", result.Repr()) }
}

func TestAddFloats(t *testing.T) {
	result, err := evalWithStdlib("(+ 1.5 2.5)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.FloatNode(4.0)) { t.Errorf("expected 4, got %s", result.Repr()) }
}

func TestAddMixed(t *testing.T) {
	result, err := evalWithStdlib("(+ 1 2.5)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.FloatNode(3.5)) { t.Errorf("expected 3.5, got %s", result.Repr()) }
}

func TestAddMixedReverse(t *testing.T) {
	result, err := evalWithStdlib("(+ 2.5 1)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.FloatNode(3.5)) { t.Errorf("expected 3.5, got %s", result.Repr()) }
}

func TestSubtract(t *testing.T) {
	result, err := evalWithStdlib("(- 10 3)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.IntNode(7)) { t.Errorf("expected 7, got %s", result.Repr()) }
}

func TestSubtractFloat(t *testing.T) {
	result, err := evalWithStdlib("(- 10.5 3.0)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.FloatNode(7.5)) { t.Errorf("expected 7.5, got %s", result.Repr()) }
}

func TestMultiply(t *testing.T) {
	result, err := evalWithStdlib("(* 3 4)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.IntNode(12)) { t.Errorf("expected 12, got %s", result.Repr()) }
}

func TestMultiplyFloat(t *testing.T) {
	result, err := evalWithStdlib("(* 2.5 4.0)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.FloatNode(10.0)) { t.Errorf("expected 10, got %s", result.Repr()) }
}

func TestDivideIntegers(t *testing.T) {
	result, err := evalWithStdlib("(/ 10 3)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.IntNode(3)) { t.Errorf("expected 3 (integer division), got %s", result.Repr()) }
}

func TestDivideFloats(t *testing.T) {
	result, err := evalWithStdlib("(/ 10.0 4.0)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.FloatNode(2.5)) { t.Errorf("expected 2.5, got %s", result.Repr()) }
}

func TestDivideByZero(t *testing.T) {
	_, err := evalWithStdlib("(/ 10 0)")
	if err == nil { t.Fatal("expected division by zero error") }
	if !strings.Contains(err.Error(), "division by zero") { t.Errorf("got: %v", err) }
}

func TestDivideByZeroFloat(t *testing.T) {
	_, err := evalWithStdlib("(/ 10.0 0.0)")
	if err == nil { t.Fatal("expected division by zero error") }
}

func TestModIntegers(t *testing.T) {
	result, err := evalWithStdlib("(mod 10 3)")
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.IntNode(1)) { t.Errorf("expected 1, got %s", result.Repr()) }
}

func TestModByZero(t *testing.T) {
	_, err := evalWithStdlib("(mod 10 0)")
	if err == nil { t.Fatal("expected division by zero error") }
}

func TestModFloatErrors(t *testing.T) {
	_, err := evalWithStdlib("(mod 10.0 3.0)")
	if err == nil { t.Fatal("expected error for mod on floats") }
}

func TestArithmeticTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(+ "a" 1)`)
	if err == nil { t.Fatal("expected type error") }
	if !strings.Contains(err.Error(), "expected number") { t.Errorf("got: %v", err) }
}

func TestArithmeticSecondArgTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(+ 1 "b")`)
	if err == nil { t.Fatal("expected type error") }
	if !strings.Contains(err.Error(), "expected number") { t.Errorf("got: %v", err) }
}
