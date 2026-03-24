package mummy

import (
	"testing"
)

func TestRegisterAndLookupSarcophagus(t *testing.T) {
	defer ClearRegistry()

	entry := &SarcophagusEntry{
		Names:    []string{"add", "subtract"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {},
	}
	RegisterSarcophagus("testpkg", "github.com/example/testpkg", entry)

	found := LookupSarcophagus("testpkg")
	if found == nil {
		t.Fatal("expected to find sarcophagus by short name")
	}
	if len(found.Names) != 2 {
		t.Errorf("expected 2 names, got %d", len(found.Names))
	}
}

func TestLookupByFullPath(t *testing.T) {
	defer ClearRegistry()

	entry := &SarcophagusEntry{
		Names:    []string{"add"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {},
	}
	RegisterSarcophagus("testpkg", "github.com/example/testpkg", entry)

	found := LookupSarcophagus("github.com/example/testpkg")
	if found == nil {
		t.Fatal("expected to find sarcophagus by full path")
	}
}

func TestLookupNonexistent(t *testing.T) {
	defer ClearRegistry()

	found := LookupSarcophagus("nonexistent")
	if found != nil {
		t.Error("expected nil for nonexistent sarcophagus")
	}
}

func TestRegisterWithPrefixAddsPrefix(t *testing.T) {
	defer ClearRegistry()

	registered := map[string]interface{}{}
	register := func(name string, fn interface{}) {
		registered[name] = fn
	}

	entry := &SarcophagusEntry{
		Names: []string{"add", "subtract"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {
			RegisterIfAllowed(prefix, only, "add", "dummy", register)
			RegisterIfAllowed(prefix, only, "subtract", "dummy", register)
		},
	}

	entry.Register("pkg", nil, register)

	if _, ok := registered["pkg:add"]; !ok {
		t.Error("expected pkg:add to be registered")
	}
	if _, ok := registered["pkg:subtract"]; !ok {
		t.Error("expected pkg:subtract to be registered")
	}
}

func TestRegisterWithOnlyFilters(t *testing.T) {
	defer ClearRegistry()

	registered := map[string]interface{}{}
	register := func(name string, fn interface{}) {
		registered[name] = fn
	}

	only := map[string]bool{"add": true}

	entry := &SarcophagusEntry{
		Names: []string{"add", "subtract"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {
			RegisterIfAllowed(prefix, only, "add", "dummy", register)
			RegisterIfAllowed(prefix, only, "subtract", "dummy", register)
		},
	}

	entry.Register("pkg", only, register)

	if _, ok := registered["pkg:add"]; !ok {
		t.Error("expected pkg:add to be registered")
	}
	if _, ok := registered["pkg:subtract"]; ok {
		t.Error("expected pkg:subtract to NOT be registered")
	}
}

func TestRegisterDuplicateShortName(t *testing.T) {
	defer ClearRegistry()

	entry1 := &SarcophagusEntry{Names: []string{"add"}, Register: func(string, map[string]bool, func(string, interface{})) {}}
	entry2 := &SarcophagusEntry{Names: []string{"mul"}, Register: func(string, map[string]bool, func(string, interface{})) {}}

	RegisterSarcophagus("pkg", "example/pkg1", entry1)
	RegisterSarcophagus("pkg", "example/pkg2", entry2)

	// Last registration wins for short name
	if LookupSarcophagus("pkg") != entry2 {
		t.Error("expected last registration to win for short name")
	}

	// Full paths should still work
	if LookupSarcophagus("example/pkg1") != entry1 {
		t.Error("expected first entry by full path")
	}
}

func TestClearRegistry(t *testing.T) {
	entry := &SarcophagusEntry{Names: []string{"x"}, Register: func(string, map[string]bool, func(string, interface{})) {}}
	RegisterSarcophagus("temp", "example/temp", entry)

	if LookupSarcophagus("temp") == nil {
		t.Fatal("expected entry before clear")
	}

	ClearRegistry()

	if LookupSarcophagus("temp") != nil {
		t.Error("expected nil after clear")
	}
}
