package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	e "github.com/archevel/ghoul/expressions"
)

type ModuleExports struct {
	Names    []string
	Bindings map[string]e.Expr
}

type ModuleState struct {
	currentFile string
	loading     []string
	loadingSet  map[string]bool
	loaded      map[string]*ModuleExports
}

func NewModuleState(currentFile string) *ModuleState {
	return &ModuleState{
		currentFile: currentFile,
		loading:     nil,
		loadingSet:  map[string]bool{},
		loaded:      map[string]*ModuleExports{},
	}
}

func (ms *ModuleState) CheckCycle(path string) error {
	if ms.loadingSet[path] {
		chain := append(ms.loading, path)
		return fmt.Errorf("circular dependency: %s", strings.Join(chain, " → "))
	}
	return nil
}

func (ms *ModuleState) BeginLoading(path string) {
	ms.loading = append(ms.loading, path)
	ms.loadingSet[path] = true
}

func (ms *ModuleState) FinishLoading(path string) {
	delete(ms.loadingSet, path)
	if len(ms.loading) > 0 && ms.loading[len(ms.loading)-1] == path {
		ms.loading = ms.loading[:len(ms.loading)-1]
	}
}

func (ms *ModuleState) CacheModule(path string, exports *ModuleExports) {
	ms.loaded[path] = exports
}

func (ms *ModuleState) GetCached(path string) *ModuleExports {
	return ms.loaded[path]
}

func (ms *ModuleState) ResolveFile(name string) (string, error) {
	if ms.currentFile == "" {
		return "", fmt.Errorf("cannot require Ghoul modules from REPL (no file context)")
	}

	dir := filepath.Dir(ms.currentFile)
	path := filepath.Join(dir, name+".ghl")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("module not found: %s (searched for %s)", name, path)
	}

	return path, nil
}

// ForChild creates a new ModuleState for evaluating a child module,
// sharing the loading state and cache.
func (ms *ModuleState) ForChild(childFile string) *ModuleState {
	return &ModuleState{
		currentFile: childFile,
		loading:     ms.loading,
		loadingSet:  ms.loadingSet,
		loaded:      ms.loaded,
	}
}
