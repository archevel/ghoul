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

func TestSyntaxToDatumInMacro(t *testing.T) {
	// A general transformer receives SyntaxObjects — syntax->datum should unwrap them
	result, err := evalWithStdlib(`
(define-syntax get-datum
  (lambda (stx)
    (syntax->datum (car (cdr stx)))))
(get-datum 42)
`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Integer(42)) { t.Errorf("expected 42, got %s", result.Repr()) }
}

func TestIdentifierPredicateInMacro(t *testing.T) {
	// identifier? should return #t for syntax objects wrapping identifiers
	result, err := evalWithStdlib(`
(define-syntax is-id
  (lambda (stx)
    (identifier? (car (cdr stx)))))
(is-id foo)
`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Boolean(true)) { t.Errorf("expected #t, got %s", result.Repr()) }
}

func TestIdentifierPredicateNonIdInMacro(t *testing.T) {
	result, err := evalWithStdlib(`
(define-syntax is-id
  (lambda (stx)
    (identifier? (car (cdr stx)))))
(is-id 42)
`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Boolean(false)) { t.Errorf("expected #f, got %s", result.Repr()) }
}

func TestDatumToSyntaxInMacro(t *testing.T) {
	// datum->syntax with a syntax object context should copy marks
	result, err := evalWithStdlib(`
(define-syntax make-val
  (lambda (stx)
    (define ctx (car (cdr stx)))
    (datum->syntax ctx 99)))
(make-val placeholder)
`)
	if err != nil { t.Fatal(err) }
	if !result.Equiv(e.Integer(99)) { t.Errorf("expected 99, got %s", result.Repr()) }
}

func TestDatumToSyntax(t *testing.T) {
	// datum->syntax with a non-syntax context just wraps
	result, err := evalWithStdlib(`(datum->syntax 42 'foo)`)
	if err != nil { t.Fatal(err) }
	if result == nil { t.Fatal("expected non-nil result") }
}
