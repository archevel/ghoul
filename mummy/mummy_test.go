package mummy

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestMummyImplementsExpr(t *testing.T) {
	var expr e.Expr = Entomb(42, "int")
	if expr == nil {
		t.Fatal("Mummy should implement Expr")
	}
}

func TestEntombAndUnwrap(t *testing.T) {
	m := Entomb("hello", "string")
	if m.Unwrap() != "hello" {
		t.Errorf("expected 'hello', got %v", m.Unwrap())
	}
}

func TestMummyRepr(t *testing.T) {
	m := Entomb(42, "int")
	if m.Repr() != "#<mummy:int>" {
		t.Errorf("expected '#<mummy:int>', got '%s'", m.Repr())
	}
}

func TestMummyReprWithStructType(t *testing.T) {
	type Person struct{ Name string }
	m := Entomb(&Person{Name: "Alice"}, "*Person")
	if m.Repr() != "#<mummy:*Person>" {
		t.Errorf("expected '#<mummy:*Person>', got '%s'", m.Repr())
	}
}

func TestMummyEquivSameWrappedComparableValue(t *testing.T) {
	m1 := Entomb(42, "int")
	m2 := Entomb(42, "int")
	if !m1.Equiv(m2) {
		t.Error("mummies wrapping equal comparable values should be Equiv")
	}
}

func TestMummyEquivSamePointer(t *testing.T) {
	x := 42
	m1 := Entomb(&x, "*int")
	m2 := Entomb(&x, "*int")
	if !m1.Equiv(m2) {
		t.Error("mummies wrapping the same pointer should be Equiv")
	}
}

func TestMummyEquivUncomparableTypesReturnFalse(t *testing.T) {
	val := []int{1, 2, 3}
	m1 := Entomb(val, "[]int")
	m2 := Entomb(val, "[]int")
	// Slices are not comparable in Go, so Equiv returns false
	// rather than panicking
	if m1.Equiv(m2) {
		t.Error("uncomparable types should not be Equiv (Go semantics)")
	}
}

func TestMummyEquivDifferentValues(t *testing.T) {
	m1 := Entomb(42, "int")
	m2 := Entomb(99, "int")
	if m1.Equiv(m2) {
		t.Error("mummies wrapping different values should not be Equiv")
	}
}

func TestMummyEquivWithNonMummy(t *testing.T) {
	m := Entomb(42, "int")
	if m.Equiv(e.Integer(42)) {
		t.Error("mummy should not be Equiv to a non-mummy")
	}
}

func TestMummyEquivNilWrapped(t *testing.T) {
	m1 := Entomb(nil, "nil")
	m2 := Entomb(nil, "nil")
	if !m1.Equiv(m2) {
		t.Error("mummies wrapping nil should be Equiv")
	}
}

func TestEntombWithPointer(t *testing.T) {
	x := 42
	m := Entomb(&x, "*int")
	ptr, ok := m.Unwrap().(*int)
	if !ok {
		t.Fatal("expected *int")
	}
	if *ptr != 42 {
		t.Errorf("expected 42, got %d", *ptr)
	}
}
