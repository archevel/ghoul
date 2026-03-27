package consume

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/mummy"
	p "github.com/archevel/ghoul/exhumer"
)

func registerTestModule() {
	dummyFunc := func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.IntNode(42), nil
	}
	mummy.RegisterSarcophagus("testmod", "github.com/example/testmod", &mummy.SarcophagusEntry{
		Names: []string{"foo", "bar"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {
			mummy.RegisterIfAllowed(prefix, only, "foo", dummyFunc, register)
			mummy.RegisterIfAllowed(prefix, only, "bar", dummyFunc, register)
		},
	})
}

func TestRequireBasic(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	env := NewEnvironment()

	in := `(require testmod) (testmod:foo)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	result, err := Evaluate(parsed.Expressions, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireWithAlias(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	env := NewEnvironment()

	in := `(require testmod as tm) (tm:foo)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	result, err := Evaluate(parsed.Expressions, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireWithOnly(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	env := NewEnvironment()

	in := `(require testmod only (foo)) (testmod:foo)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	result, err := Evaluate(parsed.Expressions, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireWithOnlyFiltersOut(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	env := NewEnvironment()

	in := `(require testmod only (foo)) (testmod:bar)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error — bar should not be registered")
	}
	if !strings.Contains(err.Error(), "undefined identifier") {
		t.Errorf("expected undefined identifier error, got: %v", err)
	}
}

func TestRequireWithAliasAndOnly(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	env := NewEnvironment()

	in := `(require testmod as tm only (bar)) (tm:bar)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	result, err := Evaluate(parsed.Expressions, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireNonexistentModule(t *testing.T) {
	defer mummy.ClearRegistry()
	env := NewEnvironment()

	in := `(require nonexistent)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for nonexistent module")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected module name in error, got: %v", err)
	}
}

func TestRequireNameClash(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()

	dummyFunc2 := func(args []*e.Node, ev *Evaluator) (*e.Node, error) {
		return e.IntNode(99), nil
	}
	mummy.RegisterSarcophagus("othermod", "github.com/example/othermod", &mummy.SarcophagusEntry{
		Names: []string{"foo"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {
			mummy.RegisterIfAllowed(prefix, only, "foo", dummyFunc2, register)
		},
	})

	env := NewEnvironment()
	in := `(require testmod as x) (require othermod as x)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for name clash")
	}
	if !strings.Contains(err.Error(), "already defined") {
		t.Errorf("expected 'already defined' in error, got: %v", err)
	}
}

func TestRequireSameModuleTwice(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	env := NewEnvironment()

	in := `(require testmod) (require testmod) (testmod:foo)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	result, err := Evaluate(parsed.Expressions, env)
	if err != nil {
		t.Fatalf("requiring same module twice should work, got: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireSameModuleDifferentAlias(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	env := NewEnvironment()

	in := `(require testmod as a) (require testmod as b) (a:foo)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	result, err := Evaluate(parsed.Expressions, env)
	if err != nil {
		t.Fatalf("same module under different aliases should work, got: %v", err)
	}
	if !result.Equiv(e.IntNode(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestRequireEmptyForm(t *testing.T) {
	defer mummy.ClearRegistry()
	env := NewEnvironment()

	in := `(require)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for empty require")
	}
}

func TestRequireNonIdentifierModuleName(t *testing.T) {
	env := NewEnvironment()

	in := `(require 42)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for non-identifier module name")
	}
	if !strings.Contains(err.Error(), "module name must be an identifier") {
		t.Errorf("expected 'module name must be an identifier' in error, got: %v", err)
	}
}

func TestRequireAsMissingAlias(t *testing.T) {
	env := NewEnvironment()

	in := `(require somemod as)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for missing alias after 'as'")
	}
	if !strings.Contains(err.Error(), "expected alias after 'as'") {
		t.Errorf("expected 'expected alias after as' in error, got: %v", err)
	}
}

func TestRequireAsNonIdentifierAlias(t *testing.T) {
	env := NewEnvironment()

	in := `(require somemod as 42)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for non-identifier alias")
	}
	if !strings.Contains(err.Error(), "alias must be an identifier") {
		t.Errorf("expected 'alias must be an identifier' in error, got: %v", err)
	}
}

func TestRequireOnlyMissingList(t *testing.T) {
	env := NewEnvironment()

	in := `(require somemod only)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for missing name list after 'only'")
	}
	if !strings.Contains(err.Error(), "expected name list after 'only'") {
		t.Errorf("expected 'expected name list' in error, got: %v", err)
	}
}

func TestRequireOnlyNonList(t *testing.T) {
	env := NewEnvironment()

	in := `(require somemod only foo)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error when 'only' is followed by non-list")
	}
	if !strings.Contains(err.Error(), "'only' must be followed by a list") {
		t.Errorf("expected 'only must be followed by a list' in error, got: %v", err)
	}
}

func TestRequireOnlyListWithNonIdentifier(t *testing.T) {
	env := NewEnvironment()

	in := `(require somemod only (42))`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for non-identifier in 'only' list")
	}
	if !strings.Contains(err.Error(), "'only' list must contain identifiers") {
		t.Errorf("expected 'only list must contain identifiers' in error, got: %v", err)
	}
}

func TestRequireUnexpectedKeyword(t *testing.T) {
	env := NewEnvironment()

	in := `(require somemod with stuff)`
	r := strings.NewReader(in)
	_, parsed := p.Parse(r)
	_, err := Evaluate(parsed.Expressions, env)
	if err == nil {
		t.Fatal("expected error for unexpected keyword")
	}
	if !strings.Contains(err.Error(), "unexpected keyword 'with'") {
		t.Errorf("expected 'unexpected keyword' in error, got: %v", err)
	}
}
