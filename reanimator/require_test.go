package reanimator

import (
	"strings"
	"testing"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
	"github.com/archevel/ghoul/mummy"
)

func registerTestModule() {
	dummyFunc := func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
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

func reanimateAndLookup(t *testing.T, input string) (*e.Node, error) {
	t.Helper()
	r := newTestReanimator()
	nodes := parseNodes(t, input)
	_, err := r.ReanimateNodes(nodes)
	if err != nil {
		return nil, err
	}
	return r.evalEnv.LookupByName("testmod:foo")
}

func TestRequireBasic(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod)`)
	_, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, err := r.evalEnv.LookupByName("testmod:foo")
	if err != nil {
		t.Fatalf("testmod:foo not found: %v", err)
	}
	if val.FuncVal == nil {
		t.Error("expected function value")
	}
}

func TestRequireWithAlias(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod as tm)`)
	_, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, err := r.evalEnv.LookupByName("tm:foo")
	if err != nil {
		t.Fatalf("tm:foo not found: %v", err)
	}
	if val.FuncVal == nil {
		t.Error("expected function value")
	}
}

func TestRequireWithOnly(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod only (foo))`)
	_, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := r.evalEnv.LookupByName("testmod:foo"); err != nil {
		t.Fatal("testmod:foo should be registered")
	}
}

func TestRequireWithOnlyFiltersOut(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod only (foo))`)
	_, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := r.evalEnv.LookupByName("testmod:bar"); err == nil {
		t.Fatal("testmod:bar should NOT be registered")
	}
}

func TestRequireWithAliasAndOnly(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod as tm only (bar))`)
	_, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := r.evalEnv.LookupByName("tm:bar"); err != nil {
		t.Fatal("tm:bar should be registered")
	}
}

func TestRequireNonexistentModule(t *testing.T) {
	defer mummy.ClearRegistry()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require nonexistent)`)
	_, err := r.ReanimateNodes(nodes)
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

	dummyFunc2 := func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		return e.IntNode(99), nil
	}
	mummy.RegisterSarcophagus("othermod", "github.com/example/othermod", &mummy.SarcophagusEntry{
		Names: []string{"foo"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {
			mummy.RegisterIfAllowed(prefix, only, "foo", dummyFunc2, register)
		},
	})

	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod as x) (require othermod as x)`)
	_, err := r.ReanimateNodes(nodes)
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
	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod) (require testmod)`)
	_, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("requiring same module twice should work, got: %v", err)
	}
}

func TestRequireSameModuleDifferentAlias(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod as a) (require testmod as b)`)
	_, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("same module under different aliases should work, got: %v", err)
	}
	if _, err := r.evalEnv.LookupByName("a:foo"); err != nil {
		t.Fatal("a:foo should be registered")
	}
	if _, err := r.evalEnv.LookupByName("b:foo"); err != nil {
		t.Fatal("b:foo should be registered")
	}
}

func TestRequireEmptyForm(t *testing.T) {
	defer mummy.ClearRegistry()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for empty require")
	}
}

func TestRequireNonIdentifierModuleName(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(require 42)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-identifier module name")
	}
	if !strings.Contains(err.Error(), "module name must be an identifier") {
		t.Errorf("expected 'module name must be an identifier' in error, got: %v", err)
	}
}

func TestRequireAsMissingAlias(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(require somemod as)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for missing alias after 'as'")
	}
	if !strings.Contains(err.Error(), "expected alias after 'as'") {
		t.Errorf("expected 'expected alias after as' in error, got: %v", err)
	}
}

func TestRequireAsNonIdentifierAlias(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(require somemod as 42)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-identifier alias")
	}
	if !strings.Contains(err.Error(), "alias must be an identifier") {
		t.Errorf("expected 'alias must be an identifier' in error, got: %v", err)
	}
}

func TestRequireOnlyMissingList(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(require somemod only)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for missing name list after 'only'")
	}
	if !strings.Contains(err.Error(), "expected name list after 'only'") {
		t.Errorf("expected 'expected name list' in error, got: %v", err)
	}
}

func TestRequireOnlyNonList(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(require somemod only foo)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error when 'only' is followed by non-list")
	}
	if !strings.Contains(err.Error(), "'only' must be followed by a list") {
		t.Errorf("expected 'only must be followed by a list' in error, got: %v", err)
	}
}

func TestRequireOnlyListWithNonIdentifier(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(require somemod only (42))`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for non-identifier in 'only' list")
	}
	if !strings.Contains(err.Error(), "'only' list must contain identifiers") {
		t.Errorf("expected 'only list must contain identifiers' in error, got: %v", err)
	}
}

func TestRequireUnexpectedKeyword(t *testing.T) {
	r := newTestReanimator()
	nodes := parseNodes(t, `(require somemod with stuff)`)
	_, err := r.ReanimateNodes(nodes)
	if err == nil {
		t.Fatal("expected error for unexpected keyword")
	}
	if !strings.Contains(err.Error(), "unexpected keyword 'with'") {
		t.Errorf("expected 'unexpected keyword' in error, got: %v", err)
	}
}

func TestRequireStripsFromOutput(t *testing.T) {
	defer mummy.ClearRegistry()
	registerTestModule()
	r := newTestReanimator()
	nodes := parseNodes(t, `(require testmod) (+ 1 2)`)
	result, err := r.ReanimateNodes(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// require should be stripped, only (+ 1 2) remains
	if len(result) != 1 {
		t.Fatalf("expected 1 result (require stripped), got %d", len(result))
	}
	if result[0].Kind != e.CallNode {
		t.Errorf("expected CallNode, got %d", result[0].Kind)
	}
}
