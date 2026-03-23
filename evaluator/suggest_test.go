package evaluator

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	p "github.com/archevel/ghoul/parser"
)

func TestLevenshteinDistance(t *testing.T) {
	cases := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"print", "printl", 1},
		{"println", "printl", 1},
		{"println", "pritnln", 2},
		{"callback", "callbak", 1},
		{"set!", "set", 1},
		{"define", "defin", 1},
		{"define", "defne", 1},
		{"println", "display", 6},
		{"recover-from-error", "recover_from_error", 2},
	}
	for _, c := range cases {
		result := levenshteinDistance(c.a, c.b)
		if result != c.expected {
			t.Errorf("levenshtein(%q, %q) = %d, expected %d", c.a, c.b, result, c.expected)
		}
	}
}

func TestSuggestIdentifiersFindsClosestMatch(t *testing.T) {
	env := NewEnvironment()
	bindIdentifier(e.Identifier("println"), e.Integer(1), env)
	bindIdentifier(e.Identifier("print"), e.Integer(2), env)

	suggestions := suggestIdentifiers("printl", env)
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d: %v", len(suggestions), suggestions)
	}
}

func TestSuggestIdentifiersReturnsNilForDistantNames(t *testing.T) {
	env := NewEnvironment()
	bindIdentifier(e.Identifier("callback"), e.Integer(1), env)

	suggestions := suggestIdentifiers("xyz", env)
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for 'xyz', got %v", suggestions)
	}
}

func TestSuggestIdentifiersOnlyShowsMinimumDistance(t *testing.T) {
	env := NewEnvironment()
	bindIdentifier(e.Identifier("define"), e.Integer(1), env)
	bindIdentifier(e.Identifier("begin"), e.Integer(2), env)

	// "defin" is distance 1 from "define", distance 2 from "begin"
	suggestions := suggestIdentifiers("defin", env)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d: %v", len(suggestions), suggestions)
	}
	if suggestions[0] != "define" {
		t.Errorf("expected 'define', got '%s'", suggestions[0])
	}
}

func TestSuggestIdentifiersInnerScopeFirst(t *testing.T) {
	env := NewEnvironment()
	bindIdentifier(e.Identifier("foo"), e.Integer(1), env)

	// Add inner scope with "fool"
	inner := newEnvWithEmptyScope(env)
	bindIdentifier(e.Identifier("fool"), e.Integer(2), inner)

	// Both are distance 1 from "fooo", but "fool" is in the closer scope
	suggestions := suggestIdentifiers("fooo", inner)
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d: %v", len(suggestions), suggestions)
	}
	if suggestions[0] != "fool" {
		t.Errorf("expected 'fool' first (inner scope), got '%s'", suggestions[0])
	}
	if suggestions[1] != "foo" {
		t.Errorf("expected 'foo' second (outer scope), got '%s'", suggestions[1])
	}
}

func TestSuggestIdentifiersCapsAtThree(t *testing.T) {
	env := NewEnvironment()
	bindIdentifier(e.Identifier("aa"), e.Integer(1), env)
	bindIdentifier(e.Identifier("ab"), e.Integer(2), env)
	bindIdentifier(e.Identifier("ac"), e.Integer(3), env)
	bindIdentifier(e.Identifier("ad"), e.Integer(4), env)

	// All are distance 1 from "a"
	suggestions := suggestIdentifiers("a", env)
	if len(suggestions) > 3 {
		t.Errorf("expected at most 3 suggestions, got %d: %v", len(suggestions), suggestions)
	}
}

func TestSuggestIdentifiersIgnoresMarksOnScopedIdentifiers(t *testing.T) {
	env := NewEnvironment()
	si := e.ScopedIdentifier{Name: "println", Marks: map[uint64]bool{1: true}}
	bindIdentifier(si, e.Integer(1), env)

	suggestions := suggestIdentifiers("printl", env)
	if len(suggestions) != 1 || suggestions[0] != "println" {
		t.Errorf("expected ['println'], got %v", suggestions)
	}
}

func TestUndefinedIdentifierErrorIncludesSuggestion(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.NIL, nil
	})
	env.Register("println", func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.NIL, nil
	})

	r := strings.NewReader("(printl 1)")
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "did you mean") {
		t.Errorf("expected suggestion in error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "println") {
		t.Errorf("expected 'println' in suggestion, got: %s", err.Error())
	}
}

func TestUndefinedIdentifierNoSuggestionWhenNothingClose(t *testing.T) {
	env := NewEnvironment()
	env.Register("println", func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.NIL, nil
	})

	r := strings.NewReader("(xyzzy 1)")
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "did you mean") {
		t.Errorf("expected no suggestion for 'xyzzy', got: %s", err.Error())
	}
}

func TestUndefinedIdentifierSuggestsMacroNames(t *testing.T) {
	env := NewEnvironment()
	in := `(define-syntax my-swap (syntax-rules () ((my-swap x y) (begin (define tmp x) (set! x y) (set! y tmp)))))
(define a 1)
(define b 2)
(my-swp a b)`

	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "my-swap") {
		t.Errorf("expected suggestion 'my-swap' for macro name typo, got: %s", errMsg)
	}
}

func TestUndefinedIdentifierMultipleSuggestions(t *testing.T) {
	env := NewEnvironment()
	env.Register("print", func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.NIL, nil
	})
	env.Register("println", func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.NIL, nil
	})

	r := strings.NewReader("(printl 1)")
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "print") || !strings.Contains(errMsg, "println") {
		t.Errorf("expected both 'print' and 'println' in suggestion, got: %s", errMsg)
	}
}
