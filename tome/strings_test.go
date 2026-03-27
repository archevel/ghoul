package tome

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestStringAppend(t *testing.T) {
	result, _ := evalWithStdlib(`(string-append "hello" " " "world")`)
	if !result.Equiv(e.StrNode("hello world")) { t.Errorf("got %s", result.Repr()) }
}

func TestStringAppendEmpty(t *testing.T) {
	result, _ := evalWithStdlib(`(string-append)`)
	if !result.Equiv(e.StrNode("")) { t.Errorf("got %s", result.Repr()) }
}

func TestStringAppendTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(string-append "a" 1)`)
	if err == nil { t.Fatal("expected error") }
}

func TestStringLength(t *testing.T) {
	result, _ := evalWithStdlib(`(string-length "hello")`)
	if !result.Equiv(e.IntNode(5)) { t.Errorf("got %s", result.Repr()) }
}

func TestStringLengthEmpty(t *testing.T) {
	result, _ := evalWithStdlib(`(string-length "")`)
	if !result.Equiv(e.IntNode(0)) { t.Errorf("got %s", result.Repr()) }
}

func TestStringLengthUnicode(t *testing.T) {
	result, _ := evalWithStdlib(`(string-length "héllo")`)
	if !result.Equiv(e.IntNode(5)) { t.Errorf("expected 5 runes, got %s", result.Repr()) }
}

func TestSubstring(t *testing.T) {
	result, _ := evalWithStdlib(`(substring "hello world" 6 11)`)
	if !result.Equiv(e.StrNode("world")) { t.Errorf("got %s", result.Repr()) }
}

func TestSubstringOutOfBounds(t *testing.T) {
	_, err := evalWithStdlib(`(substring "hello" 0 10)`)
	if err == nil { t.Fatal("expected out of bounds error") }
}

func TestSubstringInvertedBounds(t *testing.T) {
	_, err := evalWithStdlib(`(substring "hello" 3 1)`)
	if err == nil { t.Fatal("expected error for inverted bounds") }
}

func TestStringRef(t *testing.T) {
	result, _ := evalWithStdlib(`(string-ref "hello" 1)`)
	if !result.Equiv(e.StrNode("e")) { t.Errorf("got %s", result.Repr()) }
}

func TestStringRefOutOfBounds(t *testing.T) {
	_, err := evalWithStdlib(`(string-ref "hi" 5)`)
	if err == nil { t.Fatal("expected out of bounds error") }
}

func TestStringContains(t *testing.T) {
	r1, _ := evalWithStdlib(`(string-contains? "hello world" "world")`)
	if !r1.Equiv(e.BoolNode(true)) { t.Error("expected #t") }
	r2, _ := evalWithStdlib(`(string-contains? "hello" "xyz")`)
	if !r2.Equiv(e.BoolNode(false)) { t.Error("expected #f") }
}

func TestStringSplit(t *testing.T) {
	result, _ := evalWithStdlib(`(string-split "a,b,c" ",")`)
	list := result
	if !list.First().Equiv(e.StrNode("a")) { t.Errorf("first should be 'a', got %s", list.First().Repr()) }
}

func TestStringUpcase(t *testing.T) {
	result, _ := evalWithStdlib(`(string-upcase "hello")`)
	if !result.Equiv(e.StrNode("HELLO")) { t.Errorf("got %s", result.Repr()) }
}

func TestStringDowncase(t *testing.T) {
	result, _ := evalWithStdlib(`(string-downcase "HELLO")`)
	if !result.Equiv(e.StrNode("hello")) { t.Errorf("got %s", result.Repr()) }
}

func TestStringToNumber(t *testing.T) {
	r1, _ := evalWithStdlib(`(string->number "42")`)
	if !r1.Equiv(e.IntNode(42)) { t.Errorf("got %s", r1.Repr()) }
	r2, _ := evalWithStdlib(`(string->number "3.14")`)
	if !r2.Equiv(e.FloatNode(3.14)) { t.Errorf("got %s", r2.Repr()) }
}

func TestStringToNumberInvalid(t *testing.T) {
	_, err := evalWithStdlib(`(string->number "abc")`)
	if err == nil { t.Fatal("expected error") }
	if !strings.Contains(err.Error(), "cannot parse") { t.Errorf("got: %v", err) }
}

func TestStringLengthTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(string-length 42)`)
	if err == nil { t.Fatal("expected type error") }
}

func TestSubstringTypeErrors(t *testing.T) {
	_, err := evalWithStdlib(`(substring 42 0 1)`)
	if err == nil { t.Fatal("expected type error for non-string") }
	_, err = evalWithStdlib(`(substring "hi" "a" 1)`)
	if err == nil { t.Fatal("expected type error for non-int start") }
	_, err = evalWithStdlib(`(substring "hi" 0 "b")`)
	if err == nil { t.Fatal("expected type error for non-int end") }
}

func TestStringRefTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(string-ref 42 0)`)
	if err == nil { t.Fatal("expected type error") }
	_, err = evalWithStdlib(`(string-ref "hi" "a")`)
	if err == nil { t.Fatal("expected type error for non-int index") }
}

func TestStringRefNegativeIndex(t *testing.T) {
	_, err := evalWithStdlib(`(string-ref "hi" -1)`)
	if err == nil { t.Fatal("expected out of bounds") }
}

func TestStringContainsTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(string-contains? 42 "x")`)
	if err == nil { t.Fatal("expected type error") }
	_, err = evalWithStdlib(`(string-contains? "x" 42)`)
	if err == nil { t.Fatal("expected type error") }
}

func TestStringSplitTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(string-split 42 ",")`)
	if err == nil { t.Fatal("expected type error") }
	_, err = evalWithStdlib(`(string-split "a,b" 42)`)
	if err == nil { t.Fatal("expected type error") }
}

func TestStringUpcaseTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(string-upcase 42)`)
	if err == nil { t.Fatal("expected type error") }
}

func TestStringDowncaseTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(string-downcase 42)`)
	if err == nil { t.Fatal("expected type error") }
}

func TestStringToNumberTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(string->number 42)`)
	if err == nil { t.Fatal("expected type error") }
}

func TestNumberToStringTypeError(t *testing.T) {
	_, err := evalWithStdlib(`(number->string "hi")`)
	if err == nil { t.Fatal("expected type error") }
}

func TestSubstringNegativeStart(t *testing.T) {
	_, err := evalWithStdlib(`(substring "hello" -1 3)`)
	if err == nil { t.Fatal("expected out of bounds") }
}

func TestNumberToString(t *testing.T) {
	r1, _ := evalWithStdlib(`(number->string 42)`)
	if !r1.Equiv(e.StrNode("42")) { t.Errorf("got %s", r1.Repr()) }
	r2, _ := evalWithStdlib(`(number->string 3.14)`)
	if !r2.Equiv(e.StrNode("3.14")) { t.Errorf("got %s", r2.Repr()) }
}
