package stdlib

import (
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

func TestSyntaxToDatum(t *testing.T) {
	result, err := evalWithStdlib(`(syntax->datum 42)`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Integer(42)) { t.Errorf("got %s", result.Repr()) }
}

func TestIdentifierPredicate(t *testing.T) {
	result, err := evalWithStdlib(`(identifier? 'foo)`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Boolean(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestIdentifierPredicateNonId(t *testing.T) {
	result, err := evalWithStdlib(`(identifier? 42)`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Boolean(false)) { t.Errorf("expected #f, got %s", result.Repr()) }
}

func TestBytesConversion(t *testing.T) {
	// bytes creates a mummy wrapping []byte
	result, err := evalWithStdlib(`(bytes "hello")`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestStringFromBytes(t *testing.T) {
	result, err := evalWithStdlib(`(string-from-bytes (bytes "world"))`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.String("world")) { t.Errorf("got %s", result.Repr()) }
}

func TestIntSliceConversion(t *testing.T) {
	result, err := evalWithStdlib(`(int-slice 1 2 3)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestFloatSliceConversion(t *testing.T) {
	result, err := evalWithStdlib(`(float-slice 1.0 2.0)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestGoNil(t *testing.T) {
	result, err := evalWithStdlib(`(go-nil)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestIntSliceEmpty(t *testing.T) {
	result, err := evalWithStdlib(`(int-slice)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil") }
}

func TestFloatSliceEmpty(t *testing.T) {
	result, err := evalWithStdlib(`(float-slice)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil") }
}

func TestSyntaxToDatumNonSyntax(t *testing.T) {
	result, err := evalWithStdlib(`(syntax->datum "hello")`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.String("hello")) { t.Errorf("got %s", result.Repr()) }
}

func TestDatumToSyntax(t *testing.T) {
	// datum->syntax with a non-syntax context just wraps
	result, err := evalWithStdlib(`(datum->syntax 42 'foo)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestSyntaxMatchBasic(t *testing.T) {
	// Match (foo 1 2) against pattern (foo x y) with no literals
	result, err := evalWithStdlib(`(syntax-match? '(foo 1 2) '(foo x y) '())`)
	if err != nil { t.Fatal(err) }
	// Should return an association list: ((x . 1) (y . 2))
	list, ok := result.(e.List)
	if !ok || list == e.NIL {
		t.Fatalf("expected non-empty association list, got %s", result.Repr())
	}
}

func TestSyntaxMatchNoMatch(t *testing.T) {
	// Pattern expects 3 args but code has 2
	result, err := evalWithStdlib(`(syntax-match? '(foo 1) '(foo x y) '())`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Boolean(false)) {
		t.Errorf("expected #f for non-match, got %s", result.Repr())
	}
}

func TestSyntaxMatchWithLiteral(t *testing.T) {
	// "arrow" is a literal — must match exactly
	result, err := evalWithStdlib(`(syntax-match? '(foo 1 arrow 2) '(foo x arrow y) '(arrow))`)
	if err != nil { t.Fatal(err) }
	list, ok := result.(e.List)
	if !ok || list == e.NIL {
		t.Fatalf("expected match, got %s", result.Repr())
	}
}

func TestSyntaxMatchLiteralMismatch(t *testing.T) {
	result, err := evalWithStdlib(`(syntax-match? '(foo 1 blah 2) '(foo x arrow y) '(arrow))`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Boolean(false)) {
		t.Errorf("expected #f when literal doesn't match, got %s", result.Repr())
	}
}

func TestSyntaxMatchWildcard(t *testing.T) {
	// _ matches anything without binding
	result, err := evalWithStdlib(`(syntax-match? '(foo 1 2) '(foo _ y) '())`)
	if err != nil { t.Fatal(err) }
	list, ok := result.(e.List)
	if !ok || list == e.NIL {
		t.Fatalf("expected match, got %s", result.Repr())
	}
	// Should only have y binding, not _
	first := list.First()
	pair, ok := first.(*e.Pair)
	if !ok {
		t.Fatalf("expected pair in alist, got %T", first)
	}
	if !pair.H.Equiv(e.Identifier("y")) {
		t.Errorf("expected binding for y, got %s", pair.H.Repr())
	}
}

func TestSyntaxMatchEmptyPattern(t *testing.T) {
	result, err := evalWithStdlib(`(syntax-match? '(foo) '(foo) '())`)
	if err != nil { t.Fatal(err) }
	// Should match with empty bindings list
	if result == e.NIL {
		return // empty list = match with no bindings, ok
	}
	list, ok := result.(e.List)
	if !ok {
		t.Fatalf("expected list or NIL, got %T: %s", result, result.Repr())
	}
	_ = list
}
