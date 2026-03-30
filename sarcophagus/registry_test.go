package sarcophagus

import (
	"testing"
)

func TestEntombAndUnearth(t *testing.T) {
	defer ClearRegistry()

	m := &Mummy{
		Names:    []string{"add", "subtract"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {},
	}
	Entomb("testpkg", "github.com/example/testpkg", m)

	found := Unearth("testpkg")
	if found == nil {
		t.Fatal("expected to find mummy by short name")
	}
	if len(found.Names) != 2 {
		t.Errorf("expected 2 names, got %d", len(found.Names))
	}
}

func TestUnearthByFullPath(t *testing.T) {
	defer ClearRegistry()

	m := &Mummy{
		Names:    []string{"add"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {},
	}
	Entomb("testpkg", "github.com/example/testpkg", m)

	found := Unearth("github.com/example/testpkg")
	if found == nil {
		t.Fatal("expected to find mummy by full path")
	}
}

func TestUnearthNonexistent(t *testing.T) {
	defer ClearRegistry()

	found := Unearth("nonexistent")
	if found != nil {
		t.Error("expected nil for nonexistent mummy")
	}
}

func TestRegisterWithPrefixAddsPrefix(t *testing.T) {
	defer ClearRegistry()

	registered := map[string]interface{}{}
	register := func(name string, fn interface{}) {
		registered[name] = fn
	}

	m := &Mummy{
		Names: []string{"add", "subtract"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {
			RegisterIfAllowed(prefix, only, "add", "dummy", register)
			RegisterIfAllowed(prefix, only, "subtract", "dummy", register)
		},
	}

	m.Register("pkg", nil, register)

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

	m := &Mummy{
		Names: []string{"add", "subtract"},
		Register: func(prefix string, only map[string]bool, register func(string, interface{})) {
			RegisterIfAllowed(prefix, only, "add", "dummy", register)
			RegisterIfAllowed(prefix, only, "subtract", "dummy", register)
		},
	}

	m.Register("pkg", only, register)

	if _, ok := registered["pkg:add"]; !ok {
		t.Error("expected pkg:add to be registered")
	}
	if _, ok := registered["pkg:subtract"]; ok {
		t.Error("expected pkg:subtract to NOT be registered")
	}
}

func TestEntombDuplicateShortName(t *testing.T) {
	defer ClearRegistry()

	m1 := &Mummy{Names: []string{"add"}, Register: func(string, map[string]bool, func(string, interface{})) {}}
	m2 := &Mummy{Names: []string{"mul"}, Register: func(string, map[string]bool, func(string, interface{})) {}}

	Entomb("pkg", "example/pkg1", m1)
	Entomb("pkg", "example/pkg2", m2)

	// Last entombment wins for short name
	if Unearth("pkg") != m2 {
		t.Error("expected last entombment to win for short name")
	}

	// Full paths should still work
	if Unearth("example/pkg1") != m1 {
		t.Error("expected first mummy by full path")
	}
}

func TestClearRegistry(t *testing.T) {
	m := &Mummy{Names: []string{"x"}, Register: func(string, map[string]bool, func(string, interface{})) {}}
	Entomb("temp", "example/temp", m)

	if Unearth("temp") == nil {
		t.Fatal("expected mummy before clear")
	}

	ClearRegistry()

	if Unearth("temp") != nil {
		t.Error("expected nil after clear")
	}
}
