package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGraveyard(t *testing.T) {
	content := `
[[embalm]]
package = "net/http"
skip_unwrappable = true

[[embalm]]
package = "github.com/foo/bar"
`
	path := writeTestFile(t, "graveyard.toml", content)
	entries, err := parseGraveyard(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Package != "net/http" {
		t.Errorf("expected 'net/http', got '%s'", entries[0].Package)
	}
	if !entries[0].SkipUnwrappable {
		t.Error("expected skip_unwrappable=true for net/http")
	}

	if entries[1].Package != "github.com/foo/bar" {
		t.Errorf("expected 'github.com/foo/bar', got '%s'", entries[1].Package)
	}
	if entries[1].SkipUnwrappable {
		t.Error("expected skip_unwrappable=false for github.com/foo/bar")
	}
}

func TestParseGraveyardEmpty(t *testing.T) {
	path := writeTestFile(t, "graveyard.toml", "")
	entries, err := parseGraveyard(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseGraveyardMissingPackage(t *testing.T) {
	content := `
[[embalm]]
skip_unwrappable = true
`
	path := writeTestFile(t, "graveyard.toml", content)
	_, err := parseGraveyard(path)
	if err == nil {
		t.Fatal("expected error for missing package field")
	}
}

func TestParseGraveyardInvalidTOML(t *testing.T) {
	path := writeTestFile(t, "graveyard.toml", "this is not valid toml [[[")
	_, err := parseGraveyard(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestParseGraveyardFileNotFound(t *testing.T) {
	_, err := parseGraveyard("/nonexistent/graveyard.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMergeWithStdlibIncluded(t *testing.T) {
	userEntries := []EmbalmEntry{
		{Package: "github.com/foo/bar"},
	}

	merged := mergeWithStdlib(userEntries, true)

	// Should have all stdlib + the user entry
	expected := len(defaultStdlib) + 1
	if len(merged) != expected {
		t.Fatalf("expected %d entries, got %d", expected, len(merged))
	}

	// Last entry should be the user's custom package
	last := merged[len(merged)-1]
	if last.Package != "github.com/foo/bar" {
		t.Errorf("expected last entry to be 'github.com/foo/bar', got '%s'", last.Package)
	}
}

func TestMergeWithStdlibUserOverride(t *testing.T) {
	// User overrides net/http to not skip unwrappable
	userEntries := []EmbalmEntry{
		{Package: "net/http", SkipUnwrappable: false},
	}

	merged := mergeWithStdlib(userEntries, true)

	// Should have all stdlib entries (net/http overridden)
	if len(merged) != len(defaultStdlib) {
		t.Fatalf("expected %d entries, got %d", len(defaultStdlib), len(merged))
	}

	// Find net/http and check it uses the user's setting
	for _, e := range merged {
		if e.Package == "net/http" {
			if e.SkipUnwrappable {
				t.Error("expected net/http skip_unwrappable=false (user override), got true")
			}
			return
		}
	}
	t.Error("net/http not found in merged entries")
}

func TestMergeWithStdlibDisabled(t *testing.T) {
	userEntries := []EmbalmEntry{
		{Package: "github.com/foo/bar"},
	}

	merged := mergeWithStdlib(userEntries, false)

	if len(merged) != 1 {
		t.Fatalf("expected 1 entry with stdlib disabled, got %d", len(merged))
	}
	if merged[0].Package != "github.com/foo/bar" {
		t.Errorf("expected 'github.com/foo/bar', got '%s'", merged[0].Package)
	}
}

func TestMergeWithStdlibNoDuplicates(t *testing.T) {
	// User lists "math" which is also in stdlib
	userEntries := []EmbalmEntry{
		{Package: "math", SkipUnwrappable: false},
		{Package: "github.com/foo/bar"},
	}

	merged := mergeWithStdlib(userEntries, true)

	// Should have stdlib + 1 extra, no duplicate math
	expected := len(defaultStdlib) + 1
	if len(merged) != expected {
		t.Fatalf("expected %d entries, got %d", expected, len(merged))
	}

	// math should appear exactly once with user's setting
	mathCount := 0
	for _, e := range merged {
		if e.Package == "math" {
			mathCount++
			if e.SkipUnwrappable {
				t.Error("expected math skip_unwrappable=false (user override)")
			}
		}
	}
	if mathCount != 1 {
		t.Errorf("expected math to appear once, appeared %d times", mathCount)
	}
}

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	return path
}
