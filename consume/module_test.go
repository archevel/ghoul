package consume

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/bones"
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
		Bindings: map[string]*e.Node{"foo": e.IntNode(42)},
	}
	ms.CacheModule("utils.ghl", exports)

	cached := ms.GetCached("utils.ghl")
	if cached == nil {
		t.Fatal("expected cached module")
	}
	if !cached.Bindings["foo"].Equiv(e.IntNode(42)) {
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
