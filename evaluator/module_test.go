package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
	"github.com/archevel/ghoul/mummy"
	p "github.com/archevel/ghoul/parser"
)

func TestModuleStateDetectsCycle(t *testing.T) {
	ms := NewModuleState("")
	ms.BeginLoading("a.ghl")
	ms.BeginLoading("b.ghl")

	err := ms.CheckCycle("a.ghl")
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "a.ghl") {
		t.Errorf("expected file names in cycle trace, got: %v", err)
	}
}

func TestModuleStateNoCycleForNewFile(t *testing.T) {
	ms := NewModuleState("")
	ms.BeginLoading("a.ghl")

	err := ms.CheckCycle("b.ghl")
	if err != nil {
		t.Errorf("expected no cycle for new file, got: %v", err)
	}
}

func TestModuleStateCaching(t *testing.T) {
	ms := NewModuleState("")
	exports := &ModuleExports{
		Names:    []string{"foo"},
		Bindings: map[string]e.Expr{"foo": e.Integer(42)},
	}
	ms.CacheModule("utils.ghl", exports)

	cached := ms.GetCached("utils.ghl")
	if cached == nil {
		t.Fatal("expected cached module")
	}
	if !cached.Bindings["foo"].Equiv(e.Integer(42)) {
		t.Error("expected foo=42 in cached exports")
	}
}

func TestModuleStateCacheMiss(t *testing.T) {
	ms := NewModuleState("")
	if ms.GetCached("nonexistent.ghl") != nil {
		t.Error("expected nil for uncached module")
	}
}

func TestModuleStateResolveFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "utils.ghl"), []byte("(define x 1)"), 0644)

	ms := NewModuleState(filepath.Join(dir, "main.ghl"))
	path, err := ms.ResolveFile("utils")
	if err != nil {
		t.Fatalf("expected to resolve utils, got: %v", err)
	}
	if !strings.HasSuffix(path, "utils.ghl") {
		t.Errorf("expected path ending in utils.ghl, got: %s", path)
	}
}

func TestModuleStateResolveFileMissing(t *testing.T) {
	dir := t.TempDir()
	ms := NewModuleState(filepath.Join(dir, "main.ghl"))
	_, err := ms.ResolveFile("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestModuleStateResolveFileNoCurrentFile(t *testing.T) {
	ms := NewModuleState("")
	_, err := ms.ResolveFile("utils")
	if err == nil {
		t.Fatal("expected error when no current file (REPL mode)")
	}
}

func TestLoadGhoulModule(t *testing.T) {
	dir := t.TempDir()
	utilsPath := filepath.Join(dir, "utils.ghl")
	os.WriteFile(utilsPath, []byte("(define x 42) (define y 99)"), 0644)

	mainPath := filepath.Join(dir, "main.ghl")
	os.WriteFile(mainPath, []byte("(require utils) (utils:x)"), 0644)

	env := NewEnvironment()
	ms := NewModuleState(mainPath)
	ev := New(logging.StandardLogger, env)
	ev.moduleState = ms

	r := strings.NewReader("(require utils) utils:x")
	_, parsed := p.Parse(r)
	result, err := ev.Evaluate(parsed.Expressions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.Integer(42)) {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestModuleIsolation(t *testing.T) {
	defer mummy.ClearRegistry()
	dir := t.TempDir()

	// utils.ghl requires a sarcophagus
	dummyFunc := func(args e.List, ev *Evaluator) (e.Expr, error) {
		return e.Integer(77), nil
	}
	mummy.RegisterSarcophagus("dummypkg", "dummypkg", &mummy.SarcophagusEntry{
		Names: []string{"magic"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {
			mummy.RegisterIfAllowed(prefix, only, "magic", dummyFunc, register)
		},
	})

	utilsPath := filepath.Join(dir, "utils.ghl")
	os.WriteFile(utilsPath, []byte("(require dummypkg) (define val (dummypkg:magic))"), 0644)

	mainPath := filepath.Join(dir, "main.ghl")

	env := NewEnvironment()
	ms := NewModuleState(mainPath)
	ev := New(logging.StandardLogger, env)
	ev.moduleState = ms

	// After requiring utils, dummypkg should NOT be in main's environment
	r := strings.NewReader("(require utils) utils:val")
	_, parsed := p.Parse(r)
	result, err := ev.Evaluate(parsed.Expressions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.Integer(77)) {
		t.Errorf("expected 77, got %s", result.Repr())
	}

	// dummypkg:magic should NOT be accessible in main
	r2 := strings.NewReader("dummypkg:magic")
	_, parsed2 := p.Parse(r2)
	_, err2 := Evaluate(parsed2.Expressions, env)
	if err2 == nil {
		t.Error("expected error — dummypkg should not be in main's scope")
	}
}

func TestCircularDependencyError(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "a.ghl"), []byte("(require b) (define x 1)"), 0644)
	os.WriteFile(filepath.Join(dir, "b.ghl"), []byte("(require a) (define y 2)"), 0644)

	mainPath := filepath.Join(dir, "main.ghl")
	env := NewEnvironment()
	ms := NewModuleState(mainPath)
	ev := New(logging.StandardLogger, env)
	ev.moduleState = ms

	r := strings.NewReader("(require a)")
	_, parsed := p.Parse(r)
	_, err := ev.Evaluate(parsed.Expressions)
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' in error, got: %v", err)
	}
}

func TestRequireSameModuleFromTwoModules(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "shared.ghl"), []byte("(define val 55)"), 0644)
	os.WriteFile(filepath.Join(dir, "a.ghl"), []byte("(require shared) (define a-val shared:val)"), 0644)
	os.WriteFile(filepath.Join(dir, "b.ghl"), []byte("(require shared) (define b-val shared:val)"), 0644)

	mainPath := filepath.Join(dir, "main.ghl")
	env := NewEnvironment()
	ms := NewModuleState(mainPath)
	ev := New(logging.StandardLogger, env)
	ev.moduleState = ms

	r := strings.NewReader("(require a) (require b) a:a-val")
	_, parsed := p.Parse(r)
	result, err := ev.Evaluate(parsed.Expressions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equiv(e.Integer(55)) {
		t.Errorf("expected 55, got %s", result.Repr())
	}
}
