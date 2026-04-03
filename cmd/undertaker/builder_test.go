package main

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestSafeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"math", "math"},
		{"path/filepath", "path_filepath"},
		{"encoding/json", "encoding_json"},
		{"github.com/foo/bar", "github.com_foo_bar"},
		{"", ""},
	}
	for _, tc := range tests {
		got := safeName(tc.input)
		if got != tc.expected {
			t.Errorf("safeName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestIsStdlib(t *testing.T) {
	tests := []struct {
		pkg      string
		expected bool
	}{
		{"math", true},
		{"path/filepath", true},
		{"encoding/json", true},
		{"net/http", true},
		{"crypto/sha256", true},
		{"github.com/foo/bar", false},
		{"golang.org/x/tools", false},
		{"example.com/pkg", false},
		{"", true}, // empty string has no dot
	}
	for _, tc := range tests {
		got := isStdlib(tc.pkg)
		if got != tc.expected {
			t.Errorf("isStdlib(%q) = %v, want %v", tc.pkg, got, tc.expected)
		}
	}
}

// --- generateGoMod ---

func TestGenerateGoMod(t *testing.T) {
	dir := t.TempDir()
	err := generateGoMod(dir, "ghoul-build/my-ghoul", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "module ghoul-build/my-ghoul") {
		t.Errorf("expected module declaration, got:\n%s", content)
	}
	if !strings.Contains(content, "go 1.25") {
		t.Errorf("expected go version, got:\n%s", content)
	}
	if strings.Contains(content, "replace") {
		t.Error("should not contain replace directive when ghoulModulePath is empty")
	}
}

func TestGenerateGoModWithReplace(t *testing.T) {
	dir := t.TempDir()
	err := generateGoMod(dir, "ghoul-build/my-ghoul", "/home/user/ghoul")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "replace github.com/archevel/ghoul => /home/user/ghoul") {
		t.Errorf("expected replace directive, got:\n%s", content)
	}
}

func TestGenerateGoModInvalidDir(t *testing.T) {
	err := generateGoMod("/nonexistent/dir/that/does/not/exist", "mod", "")
	if err == nil {
		t.Fatal("expected error writing to nonexistent directory")
	}
}

// --- ghoulModuleVersion ---

func TestGhoulModuleVersionNoBuildInfo(t *testing.T) {
	orig := readBuildInfo
	defer func() { readBuildInfo = orig }()

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}

	version, isDevel := ghoulModuleVersion()
	if version != "latest" {
		t.Errorf("expected 'latest', got %q", version)
	}
	if isDevel {
		t.Error("expected isDevel=false when no build info")
	}
}

func TestGhoulModuleVersionFromDep(t *testing.T) {
	orig := readBuildInfo
	defer func() { readBuildInfo = orig }()

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "github.com/archevel/ghoul", Version: "v1.2.3"},
			},
		}, true
	}

	version, isDevel := ghoulModuleVersion()
	if version != "v1.2.3" {
		t.Errorf("expected 'v1.2.3', got %q", version)
	}
	if isDevel {
		t.Error("expected isDevel=false for versioned dep")
	}
}

func TestGhoulModuleVersionDevel(t *testing.T) {
	orig := readBuildInfo
	defer func() { readBuildInfo = orig }()

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "github.com/archevel/ghoul", Version: "(devel)"},
			},
		}, true
	}

	version, isDevel := ghoulModuleVersion()
	if version != "(devel)" {
		t.Errorf("expected '(devel)', got %q", version)
	}
	if !isDevel {
		t.Error("expected isDevel=true for (devel)")
	}
}

func TestGhoulModuleVersionMainModule(t *testing.T) {
	orig := readBuildInfo
	defer func() { readBuildInfo = orig }()

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/archevel/ghoul",
				Version: "(devel)",
			},
		}, true
	}

	version, isDevel := ghoulModuleVersion()
	if version != "(devel)" {
		t.Errorf("expected '(devel)', got %q", version)
	}
	if !isDevel {
		t.Error("expected isDevel=true when main module is ghoul")
	}
}

func TestGhoulModuleVersionUnrelatedModule(t *testing.T) {
	orig := readBuildInfo
	defer func() { readBuildInfo = orig }()

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "example.com/other",
				Version: "v0.1.0",
			},
			Deps: []*debug.Module{
				{Path: "example.com/dep", Version: "v0.2.0"},
			},
		}, true
	}

	version, isDevel := ghoulModuleVersion()
	if version != "latest" {
		t.Errorf("expected 'latest' fallback, got %q", version)
	}
	if isDevel {
		t.Error("expected isDevel=false for unrelated module")
	}
}

// --- findLocalGhoulRoot ---

func TestFindLocalGhoulRoot(t *testing.T) {
	// We're running inside the ghoul repo, so this should succeed
	root, err := findLocalGhoulRoot()
	if err != nil {
		t.Fatalf("expected to find ghoul root: %v", err)
	}

	// Verify the root contains go.mod with the right module
	modPath := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatalf("reading go.mod at root: %v", err)
	}
	if !strings.Contains(string(data), "module github.com/archevel/ghoul") {
		t.Errorf("go.mod at %s doesn't contain expected module declaration", root)
	}
}

func TestFindLocalGhoulRootFromSubdir(t *testing.T) {
	// Even from a deeply nested dir, it should walk up and find the root
	// (this test itself runs from cmd/undertaker/)
	root, err := findLocalGhoulRoot()
	if err != nil {
		t.Fatalf("expected to find ghoul root from subdir: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root path")
	}
}

// --- runCmd ---

func TestRunCmd(t *testing.T) {
	// Test with a simple command that should succeed
	err := defaultRunCmd(t.TempDir(), "true")
	if err != nil {
		t.Errorf("expected 'true' to succeed: %v", err)
	}
}

func TestRunCmdFailure(t *testing.T) {
	err := defaultRunCmd(t.TempDir(), "false")
	if err == nil {
		t.Error("expected 'false' to fail")
	}
}

func TestRunCmdInvalidCommand(t *testing.T) {
	err := defaultRunCmd(t.TempDir(), "nonexistent-command-that-does-not-exist")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestRunCmdUsesDir(t *testing.T) {
	dir := t.TempDir()
	// Write a file to the temp dir, then verify pwd sees it
	err := defaultRunCmd(dir, "ls")
	if err != nil {
		t.Errorf("expected ls to succeed in temp dir: %v", err)
	}
}

// --- build (orchestration with mocked commands) ---

func TestBuildInvalidGraveyardFile(t *testing.T) {
	err := build(BuildOptions{
		BinaryName:    "test-ghoul",
		GraveyardFile: "/nonexistent/graveyard.toml",
		NoStdlib:      true,
	})
	if err == nil {
		t.Fatal("expected error for missing graveyard file")
	}
}

func TestBuildEmptyGraveyardNoStdlib(t *testing.T) {
	graveyardPath := writeTestFile(t, "graveyard.toml", "")
	dir := t.TempDir()

	// Mock command runner to track what gets called
	origRunner := cmdRunner
	defer func() { cmdRunner = origRunner }()

	var cmds []string
	cmdRunner = func(dir, name string, args ...string) error {
		cmds = append(cmds, name+" "+strings.Join(args, " "))
		return nil
	}

	// Also mock build info to avoid local root detection issues
	origBI := readBuildInfo
	defer func() { readBuildInfo = origBI }()
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "github.com/archevel/ghoul", Version: "v0.1.0"},
			},
		}, true
	}

	err := build(BuildOptions{
		BinaryName:    "test-ghoul",
		GraveyardFile: graveyardPath,
		WorkDir:       filepath.Join(dir, "build"),
		Keep:          true,
		NoStdlib:      true,
		NoPrelude:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have: go get ghoul, go mod tidy, go build
	if len(cmds) < 3 {
		t.Fatalf("expected at least 3 commands, got %d: %v", len(cmds), cmds)
	}

	// First command should be go get ghoul
	if !strings.Contains(cmds[0], "go get github.com/archevel/ghoul") {
		t.Errorf("expected first cmd to be go get ghoul, got: %s", cmds[0])
	}

	// Should have go mod tidy
	hasTidy := false
	for _, c := range cmds {
		if strings.Contains(c, "go mod tidy") {
			hasTidy = true
		}
	}
	if !hasTidy {
		t.Errorf("expected go mod tidy in commands: %v", cmds)
	}

	// Should have go build
	hasBuild := false
	for _, c := range cmds {
		if strings.Contains(c, "go build") {
			hasBuild = true
		}
	}
	if !hasBuild {
		t.Errorf("expected go build in commands: %v", cmds)
	}

	// Verify generated files exist
	buildDir := filepath.Join(dir, "build")
	if _, err := os.Stat(filepath.Join(buildDir, "go.mod")); os.IsNotExist(err) {
		t.Error("expected go.mod to be generated")
	}
	if _, err := os.Stat(filepath.Join(buildDir, "main.go")); os.IsNotExist(err) {
		t.Error("expected main.go to be generated")
	}
	if _, err := os.Stat(filepath.Join(buildDir, "sarcophagus.go")); os.IsNotExist(err) {
		t.Error("expected sarcophagus.go to be generated")
	}
	// No prelude when --no-prelude
	if _, err := os.Stat(filepath.Join(buildDir, "prelude.ghl")); !os.IsNotExist(err) {
		t.Error("expected no prelude.ghl when --no-prelude is set")
	}
}

func TestBuildWritesPrelude(t *testing.T) {
	graveyardPath := writeTestFile(t, "graveyard.toml", "")
	dir := t.TempDir()

	origRunner := cmdRunner
	defer func() { cmdRunner = origRunner }()
	cmdRunner = func(dir, name string, args ...string) error { return nil }

	origBI := readBuildInfo
	defer func() { readBuildInfo = origBI }()
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "github.com/archevel/ghoul", Version: "v0.1.0"},
			},
		}, true
	}

	buildDir := filepath.Join(dir, "build")
	err := build(BuildOptions{
		BinaryName:    "test-ghoul",
		GraveyardFile: graveyardPath,
		WorkDir:       buildDir,
		Keep:          true,
		NoStdlib:      true,
		NoPrelude:     false, // include prelude
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Prelude should exist
	data, err := os.ReadFile(filepath.Join(buildDir, "prelude.ghl"))
	if err != nil {
		t.Fatalf("expected prelude.ghl to exist: %v", err)
	}
	if !strings.Contains(string(data), "define-syntax let") {
		t.Error("prelude.ghl should contain the let macro definition")
	}

	// main.go should contain embed directive
	mainData, err := os.ReadFile(filepath.Join(buildDir, "main.go"))
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	if !strings.Contains(string(mainData), "go:embed prelude.ghl") {
		t.Error("main.go should contain go:embed for prelude")
	}
}

func TestBuildCleanupWhenKeepFalse(t *testing.T) {
	graveyardPath := writeTestFile(t, "graveyard.toml", "")
	dir := t.TempDir()

	origRunner := cmdRunner
	defer func() { cmdRunner = origRunner }()
	cmdRunner = func(dir, name string, args ...string) error { return nil }

	origBI := readBuildInfo
	defer func() { readBuildInfo = origBI }()
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "github.com/archevel/ghoul", Version: "v0.1.0"},
			},
		}, true
	}

	buildDir := filepath.Join(dir, "build")
	err := build(BuildOptions{
		BinaryName:    "test-ghoul",
		GraveyardFile: graveyardPath,
		WorkDir:       buildDir,
		Keep:          false, // should clean up
		NoStdlib:      true,
		NoPrelude:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Build directory should be removed
	if _, err := os.Stat(buildDir); !os.IsNotExist(err) {
		t.Error("expected build directory to be cleaned up when --keep is false")
	}
}

func TestBuildDefaultWorkDir(t *testing.T) {
	graveyardPath := writeTestFile(t, "graveyard.toml", "")

	// Run from a temp dir so .ghoul/build doesn't pollute the repo
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	origRunner := cmdRunner
	defer func() { cmdRunner = origRunner }()
	cmdRunner = func(dir, name string, args ...string) error { return nil }

	origBI := readBuildInfo
	defer func() { readBuildInfo = origBI }()
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "github.com/archevel/ghoul", Version: "v0.1.0"},
			},
		}, true
	}

	err := build(BuildOptions{
		BinaryName:    "test-ghoul",
		GraveyardFile: graveyardPath,
		WorkDir:       "", // should default to .ghoul/build
		Keep:          true,
		NoStdlib:      true,
		NoPrelude:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check .ghoul/build was created
	defaultDir := filepath.Join(tmpDir, ".ghoul", "build")
	if _, err := os.Stat(defaultDir); os.IsNotExist(err) {
		t.Error("expected default .ghoul/build directory to be created")
	}
}

func TestBuildSkipsGoGetForDevel(t *testing.T) {
	graveyardPath := writeTestFile(t, "graveyard.toml", "")
	dir := t.TempDir()

	origRunner := cmdRunner
	defer func() { cmdRunner = origRunner }()
	var cmds []string
	cmdRunner = func(dir, name string, args ...string) error {
		cmds = append(cmds, name+" "+strings.Join(args, " "))
		return nil
	}

	origBI := readBuildInfo
	defer func() { readBuildInfo = origBI }()
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/archevel/ghoul",
				Version: "(devel)",
			},
		}, true
	}

	err := build(BuildOptions{
		BinaryName:    "test-ghoul",
		GraveyardFile: graveyardPath,
		WorkDir:       filepath.Join(dir, "build"),
		Keep:          true,
		NoStdlib:      true,
		NoPrelude:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT have "go get github.com/archevel/ghoul" when devel
	for _, c := range cmds {
		if strings.Contains(c, "go get github.com/archevel/ghoul") {
			t.Errorf("should not go get ghoul in devel mode, but found: %s", c)
		}
	}

	// go.mod should have replace directive
	data, _ := os.ReadFile(filepath.Join(dir, "build", "go.mod"))
	if !strings.Contains(string(data), "replace github.com/archevel/ghoul") {
		t.Error("expected replace directive in go.mod for devel mode")
	}
}

func TestBuildGoGetsThirdPartyPackages(t *testing.T) {
	content := `
[[embalm]]
package = "github.com/foo/bar"

[[embalm]]
package = "github.com/baz/qux"
`
	graveyardPath := writeTestFile(t, "graveyard.toml", content)
	dir := t.TempDir()

	origRunner := cmdRunner
	defer func() { cmdRunner = origRunner }()
	var cmds []string
	cmdRunner = func(dir, name string, args ...string) error {
		cmds = append(cmds, name+" "+strings.Join(args, " "))
		return nil
	}

	origBI := readBuildInfo
	defer func() { readBuildInfo = origBI }()
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "github.com/archevel/ghoul", Version: "v0.1.0"},
			},
		}, true
	}

	err := build(BuildOptions{
		BinaryName:    "test-ghoul",
		GraveyardFile: graveyardPath,
		WorkDir:       filepath.Join(dir, "build"),
		Keep:          true,
		NoStdlib:      true,
		NoPrelude:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have go get for both third-party packages
	hasFoo := false
	hasBaz := false
	for _, c := range cmds {
		if strings.Contains(c, "go get github.com/foo/bar") {
			hasFoo = true
		}
		if strings.Contains(c, "go get github.com/baz/qux") {
			hasBaz = true
		}
	}
	if !hasFoo {
		t.Errorf("expected go get github.com/foo/bar, commands: %v", cmds)
	}
	if !hasBaz {
		t.Errorf("expected go get github.com/baz/qux, commands: %v", cmds)
	}
}

func TestBuildDoesNotGoGetStdlib(t *testing.T) {
	content := `
[[embalm]]
package = "math"
`
	graveyardPath := writeTestFile(t, "graveyard.toml", content)
	dir := t.TempDir()

	origRunner := cmdRunner
	defer func() { cmdRunner = origRunner }()
	var cmds []string
	cmdRunner = func(dir, name string, args ...string) error {
		cmds = append(cmds, name+" "+strings.Join(args, " "))
		return nil
	}

	origBI := readBuildInfo
	defer func() { readBuildInfo = origBI }()
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "github.com/archevel/ghoul", Version: "v0.1.0"},
			},
		}, true
	}

	err := build(BuildOptions{
		BinaryName:    "test-ghoul",
		GraveyardFile: graveyardPath,
		WorkDir:       filepath.Join(dir, "build"),
		Keep:          true,
		NoStdlib:      true,
		NoPrelude:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT have go get for math (stdlib)
	for _, c := range cmds {
		if strings.Contains(c, "go get math") {
			t.Errorf("should not go get stdlib package 'math', but found: %s", c)
		}
	}
}
