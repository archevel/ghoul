package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// parseGraveyard reads and decodes a graveyard.toml file.
func parseGraveyard(path string) ([]EmbalmEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading graveyard file: %w", err)
	}

	var g Graveyard
	if err := toml.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parsing graveyard TOML: %w", err)
	}

	for i, e := range g.Embalm {
		if e.Package == "" {
			return nil, fmt.Errorf("embalm entry %d: missing 'package' field", i+1)
		}
	}

	return g.Embalm, nil
}

// mergeWithStdlib prepends the default stdlib entries to the user entries.
// User entries override stdlib defaults for the same package.
func mergeWithStdlib(userEntries []EmbalmEntry, includeStdlib bool) []EmbalmEntry {
	if !includeStdlib {
		return userEntries
	}

	// Index user entries by package for override lookup
	userByPkg := make(map[string]EmbalmEntry, len(userEntries))
	for _, e := range userEntries {
		userByPkg[e.Package] = e
	}

	var merged []EmbalmEntry
	for _, stdlib := range defaultStdlib {
		if override, ok := userByPkg[stdlib.Package]; ok {
			merged = append(merged, override)
		} else {
			merged = append(merged, stdlib)
		}
	}

	// Append user entries that are not in the stdlib list
	stdlibSet := make(map[string]bool, len(defaultStdlib))
	for _, s := range defaultStdlib {
		stdlibSet[s.Package] = true
	}
	for _, e := range userEntries {
		if !stdlibSet[e.Package] {
			merged = append(merged, e)
		}
	}

	return merged
}
