package evaluator

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/mummy"
	p "github.com/archevel/ghoul/parser"
)

func registerTestModule() {
	dummyFunc := func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.Integer(42), nil
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
	if !result.Equiv(e.Integer(42)) {
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
	if !result.Equiv(e.Integer(42)) {
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
	if !result.Equiv(e.Integer(42)) {
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
	if !result.Equiv(e.Integer(42)) {
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

	dummyFunc2 := func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.Integer(99), nil
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
	if !result.Equiv(e.Integer(42)) {
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
	if !result.Equiv(e.Integer(42)) {
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
