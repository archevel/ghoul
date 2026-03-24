package mummy

import (
	e "github.com/archevel/ghoul/expressions"
)

// Registrar is the function signature for registering a Ghoul function by name.
// This avoids importing the evaluator package directly, breaking import cycles.
type Registrar func(name string, fn func(e.List, ...interface{}) (e.Expr, error))

type SarcophagusEntry struct {
	Names    []string
	Register func(prefix string, only map[string]bool, register func(string, interface{}))
}

var registry = map[string]*SarcophagusEntry{}

func RegisterSarcophagus(shortName string, fullPath string, entry *SarcophagusEntry) {
	registry[shortName] = entry
	if fullPath != shortName {
		registry[fullPath] = entry
	}
}

func LookupSarcophagus(name string) *SarcophagusEntry {
	return registry[name]
}

func ClearRegistry() {
	for k := range registry {
		delete(registry, k)
	}
}

func RegisterIfAllowed(prefix string, only map[string]bool, name string, fn interface{}, register func(string, interface{})) {
	if only != nil && !only[name] {
		return
	}
	register(prefix+":"+name, fn)
}
