package consume

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/bones"
	p "github.com/archevel/ghoul/exhumer"
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
	bindNode(e.IdentNode("println"), e.IntNode(1), env)
	bindNode(e.IdentNode("print"), e.IntNode(2), env)

	suggestions := suggestIdentifiers("printl", env)
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d: %v", len(suggestions), suggestions)
	}
}

func TestSuggestIdentifiersReturnsNilForDistantNames(t *testing.T) {
	env := NewEnvironment()
	bindNode(e.IdentNode("callback"), e.IntNode(1), env)

	suggestions := suggestIdentifiers("xyz", env)
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for 'xyz', got %v", suggestions)
	}
}

func TestSuggestIdentifiersOnlyShowsMinimumDistance(t *testing.T) {
	env := NewEnvironment()
	bindNode(e.IdentNode("define"), e.IntNode(1), env)
	bindNode(e.IdentNode("begin"), e.IntNode(2), env)

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
	bindNode(e.IdentNode("foo"), e.IntNode(1), env)

	// Add inner scope with "fool"
	inner := newEnvWithEmptyScope(env)
	bindNode(e.IdentNode("fool"), e.IntNode(2), inner)

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
	bindNode(e.IdentNode("aa"), e.IntNode(1), env)
	bindNode(e.IdentNode("ab"), e.IntNode(2), env)
	bindNode(e.IdentNode("ac"), e.IntNode(3), env)
	bindNode(e.IdentNode("ad"), e.IntNode(4), env)

	// All are distance 1 from "a"
	suggestions := suggestIdentifiers("a", env)
	if len(suggestions) > 3 {
		t.Errorf("expected at most 3 suggestions, got %d: %v", len(suggestions), suggestions)
	}
}

func TestSuggestIdentifiersIgnoresMarksOnScopedIdentifiers(t *testing.T) {
	env := NewEnvironment()
	si := e.ScopedIdentNode("println", map[uint64]bool{1: true})
	bindNode(si, e.IntNode(1), env)

	suggestions := suggestIdentifiers("printl", env)
	if len(suggestions) != 1 || suggestions[0] != "println" {
		t.Errorf("expected ['println'], got %v", suggestions)
	}
}

func TestUndefinedIdentifierErrorIncludesSuggestion(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.Nil, nil
	})
	env.Register("println", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.Nil, nil
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
	env.Register("println", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.Nil, nil
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

func TestUndefinedIdentifierMultipleSuggestions(t *testing.T) {
	env := NewEnvironment()
	env.Register("print", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.Nil, nil
	})
	env.Register("println", func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.Nil, nil
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
