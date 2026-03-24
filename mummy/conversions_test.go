package mummy

import (
	"testing"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
)

func TestBytesConvertsStringToByteSlice(t *testing.T) {
	args := e.Cons(e.String("hello"), e.NIL)
	result, err := bytesConv(args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(*Mummy)
	if !ok {
		t.Fatalf("expected *Mummy, got %T", result)
	}
	bs, ok := m.Unwrap().([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", m.Unwrap())
	}
	if string(bs) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(bs))
	}
}

func TestBytesRejectsNonString(t *testing.T) {
	args := e.Cons(e.Integer(42), e.NIL)
	_, err := bytesConv(args, nil)
	if err == nil {
		t.Error("expected error for non-string argument")
	}
}

func TestStringFromBytesConvertsBack(t *testing.T) {
	m := Entomb([]byte("world"), "[]byte")
	args := e.Cons(m, e.NIL)
	result, err := stringFromBytes(args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := result.(e.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if string(s) != "world" {
		t.Errorf("expected 'world', got '%s'", string(s))
	}
}

func TestStringFromBytesRejectsNonMummy(t *testing.T) {
	args := e.Cons(e.String("nope"), e.NIL)
	_, err := stringFromBytes(args, nil)
	if err == nil {
		t.Error("expected error for non-mummy argument")
	}
}

func TestStringFromBytesRejectsWrongMummyType(t *testing.T) {
	m := Entomb(42, "int")
	args := e.Cons(m, e.NIL)
	_, err := stringFromBytes(args, nil)
	if err == nil {
		t.Error("expected error for mummy not wrapping []byte")
	}
}

func TestIntSliceCreatesSlice(t *testing.T) {
	args := e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Cons(e.Integer(3), e.NIL)))
	result, err := intSlice(args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(*Mummy)
	s := m.Unwrap().([]int)
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", s)
	}
}

func TestIntSliceEmpty(t *testing.T) {
	result, err := intSlice(e.NIL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(*Mummy)
	s := m.Unwrap().([]int)
	if len(s) != 0 {
		t.Errorf("expected empty slice, got %v", s)
	}
}

func TestIntSliceRejectsNonInteger(t *testing.T) {
	args := e.Cons(e.Integer(1), e.Cons(e.String("bad"), e.NIL))
	_, err := intSlice(args, nil)
	if err == nil {
		t.Error("expected error for non-integer element")
	}
}

func TestFloatSliceCreatesSlice(t *testing.T) {
	args := e.Cons(e.Float(1.5), e.Cons(e.Float(2.5), e.NIL))
	result, err := floatSlice(args, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(*Mummy)
	s := m.Unwrap().([]float64)
	if len(s) != 2 || s[0] != 1.5 || s[1] != 2.5 {
		t.Errorf("expected [1.5 2.5], got %v", s)
	}
}

func TestFloatSliceRejectsNonFloat(t *testing.T) {
	args := e.Cons(e.Integer(1), e.NIL)
	_, err := floatSlice(args, nil)
	if err == nil {
		t.Error("expected error for non-float element")
	}
}

func TestGoNilCreatesNilMummy(t *testing.T) {
	result, err := goNil(e.NIL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(*Mummy)
	if !ok {
		t.Fatalf("expected *Mummy, got %T", result)
	}
	if m.Unwrap() != nil {
		t.Error("expected nil inside mummy")
	}
}

func TestRegisterConversionsDoesNotPanic(t *testing.T) {
	env := ev.NewEnvironment()
	RegisterConversions(env)
}

func TestGoNilRepr(t *testing.T) {
	result, _ := goNil(e.NIL, nil)
	m := result.(*Mummy)
	if m.Repr() != "#<mummy:nil>" {
		t.Errorf("expected '#<mummy:nil>', got '%s'", m.Repr())
	}
}
