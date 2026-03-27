package tome

import (
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestSyntaxToDatum(t *testing.T) {
	result, err := evalWithStdlib(`(syntax->datum 42)`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.IntNode(42)) { t.Errorf("got %s", result.Repr()) }
}

func TestIdentifierPredTrue(t *testing.T) {
	result, err := evalWithStdlib(`(identifier? 'x)`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.BoolNode(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestIdentifierPredFalse(t *testing.T) {
	result, err := evalWithStdlib(`(identifier? 42)`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.BoolNode(false)) { t.Errorf("expected #f, got %s", result.Repr()) }
}

func TestBytesRoundTrip(t *testing.T) {
	result, err := evalWithStdlib(`(string-from-bytes (bytes "hello"))`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.StrNode("hello")) { t.Errorf("got %s", result.Repr()) }
}

func TestGoNil(t *testing.T) {
	result, err := evalWithStdlib(`(go-nil)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestIntSlice(t *testing.T) {
	result, err := evalWithStdlib(`(int-slice 1 2 3)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestFloatSlice(t *testing.T) {
	result, err := evalWithStdlib(`(float-slice 1.5 2.5)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestDatumToSyntax(t *testing.T) {
	// datum->syntax wraps a datum with marks from the context
	result, err := evalWithStdlib(`(datum->syntax 'ctx 42)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}

func TestStringFromBytesWithStdlib(t *testing.T) {
	result, err := evalWithStdlib(`(string-from-bytes (bytes "world"))`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.StrNode("world")) { t.Errorf("got %s", result.Repr()) }
}

func TestSyntaxMatchBasic(t *testing.T) {
	result, err := evalWithStdlib(`(syntax-match? '(foo 1 2) '(foo x y) '())`)
	if err != nil { t.Fatal(err) }
	if result.Kind != e.ListNode || len(result.Children) == 0 {
		t.Fatalf("expected non-empty association list, got %s", result.Repr())
	}
}

func TestSyntaxMatchNoMatch(t *testing.T) {
	result, err := evalWithStdlib(`(syntax-match? '(foo 1) '(foo x y) '())`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.BoolNode(false)) {
		t.Errorf("expected #f for non-match, got %s", result.Repr())
	}
}

func TestSyntaxMatchWithLiteral(t *testing.T) {
	result, err := evalWithStdlib(`(syntax-match? '(foo 1 arrow 2) '(foo x arrow y) '(arrow))`)
	if err != nil { t.Fatal(err) }
	if result.Kind != e.ListNode || len(result.Children) == 0 {
		t.Fatalf("expected match, got %s", result.Repr())
	}
}

func TestSyntaxMatchLiteralMismatch(t *testing.T) {
	result, err := evalWithStdlib(`(syntax-match? '(foo 1 blah 2) '(foo x arrow y) '(arrow))`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.BoolNode(false)) {
		t.Errorf("expected #f when literal doesn't match, got %s", result.Repr())
	}
}

func TestSyntaxMatchWildcard(t *testing.T) {
	result, err := evalWithStdlib(`(syntax-match? '(foo 1 2) '(foo _ y) '())`)
	if err != nil { t.Fatal(err) }
	if result.Kind != e.ListNode || len(result.Children) == 0 {
		t.Fatalf("expected match, got %s", result.Repr())
	}
	// Should have y binding — check the first association pair
	first := result.Children[0]
	if first.Kind != e.ListNode || len(first.Children) == 0 {
		t.Fatalf("expected pair in alist, got %s", first.Repr())
	}
	if first.First().IdentName() != "y" {
		t.Errorf("expected binding for y, got %s", first.First().Repr())
	}
}

func TestSyntaxMatchEmptyPattern(t *testing.T) {
	result, err := evalWithStdlib(`(syntax-match? '(foo) '(foo) '())`)
	if err != nil { t.Fatal(err) }
	// Should match — either empty list (no bindings) or boolean false would mean no match
	if result.Kind == e.BooleanNode && !result.BoolVal {
		t.Error("expected match for matching patterns")
	}
}
