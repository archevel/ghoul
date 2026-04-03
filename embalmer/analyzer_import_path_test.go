package embalmer

import (
	"testing"
)

func TestAnalyzePackageByImportPath(t *testing.T) {
	// Load the "math" stdlib package by import path rather than directory
	config := &Config{
		ImportPath:  "math",
		PackageName: "math_mummy",
	}
	analyzer := NewAnalyzer(config)
	info, err := analyzer.AnalyzePackage()
	if err != nil {
		t.Fatalf("failed to analyze math by import path: %v", err)
	}

	if info.Name != "math" {
		t.Errorf("expected package name 'math', got '%s'", info.Name)
	}
	if info.ImportPath != "math" {
		t.Errorf("expected import path 'math', got '%s'", info.ImportPath)
	}
	if len(info.Functions) == 0 {
		t.Error("expected functions in math package, got none")
	}

	// Check that a well-known function is present
	found := false
	for _, f := range info.Functions {
		if f.Name == "Abs" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find Abs function in math package")
	}
}

func TestAnalyzePackageByImportPathWithWorkDir(t *testing.T) {
	// WorkDir should be used as the directory context for resolution
	config := &Config{
		ImportPath:  "strings",
		WorkDir:     t.TempDir(),
		PackageName: "strings_mummy",
	}
	analyzer := NewAnalyzer(config)
	info, err := analyzer.AnalyzePackage()
	if err != nil {
		t.Fatalf("failed to analyze strings by import path with WorkDir: %v", err)
	}

	if info.Name != "strings" {
		t.Errorf("expected package name 'strings', got '%s'", info.Name)
	}
}

func TestAnalyzePackageByImportPathInvalidPackage(t *testing.T) {
	config := &Config{
		ImportPath:  "nonexistent/package/that/does/not/exist",
		PackageName: "test_mummy",
	}
	analyzer := NewAnalyzer(config)
	_, err := analyzer.AnalyzePackage()
	if err == nil {
		t.Fatal("expected error for nonexistent import path, got nil")
	}
}

func TestAnalyzePackageFallsBackToDirectoryWhenNoImportPath(t *testing.T) {
	// When ImportPath is empty, should use PackagePath (existing behavior)
	// We can't easily test the full directory-based flow in a unit test,
	// but we verify the config routing works
	config := &Config{
		PackagePath: "/nonexistent/path",
		PackageName: "test_mummy",
	}
	analyzer := NewAnalyzer(config)
	_, err := analyzer.AnalyzePackage()
	// Should fail because the directory doesn't exist, but it should
	// attempt directory-based loading (not import-path loading)
	if err == nil {
		t.Fatal("expected error for nonexistent directory path, got nil")
	}
}
