package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/archevel/ghoul"
	"github.com/archevel/ghoul/embalmer"
)

// BuildOptions holds the configuration for a build invocation.
type BuildOptions struct {
	BinaryName    string
	GraveyardFile string
	WorkDir       string // build directory (default: .ghoul/build/)
	Verbose       bool
	Keep          bool // keep build directory after build
	NoPrelude     bool
	NoStdlib      bool
}

// safeName converts a Go import path to a filesystem-safe name.
func safeName(importPath string) string {
	return strings.ReplaceAll(importPath, "/", "_")
}

// ghoulModuleVersion returns the version of the ghoul module that the
// undertaker binary was built with. When running via `go run` (development),
// this returns "(devel)" and we need a replace directive instead.
func ghoulModuleVersion() (version string, isDevel bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "latest", false
	}
	for _, dep := range info.Deps {
		if dep.Path == "github.com/archevel/ghoul" {
			if dep.Version == "(devel)" {
				return dep.Version, true
			}
			return dep.Version, false
		}
	}
	// Main module is ghoul itself (running from source)
	if info.Main.Path == "github.com/archevel/ghoul" {
		return info.Main.Version, true
	}
	return "latest", false
}

// findLocalGhoulRoot tries to find the local ghoul module root by looking
// for go.mod with "module github.com/archevel/ghoul" starting from the
// undertaker binary's location.
func findLocalGhoulRoot() (string, error) {
	// Try to find it relative to the working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		modFile := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modFile); err == nil {
			if strings.Contains(string(data), "module github.com/archevel/ghoul") {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not find local ghoul module root")
}

// build runs the full build pipeline.
func build(opts BuildOptions) error {
	// Parse graveyard.toml
	entries, err := parseGraveyard(opts.GraveyardFile)
	if err != nil {
		return err
	}

	// Merge with stdlib defaults
	entries = mergeWithStdlib(entries, !opts.NoStdlib)

	if len(entries) == 0 && opts.NoStdlib {
		fmt.Println("Warning: no packages to mummify (graveyard is empty and --no-stdlib is set)")
	}

	// Set up build directory
	buildDir := opts.WorkDir
	if buildDir == "" {
		buildDir = filepath.Join(".ghoul", "build")
	}

	// Make buildDir absolute so go commands work correctly
	buildDir, err = filepath.Abs(buildDir)
	if err != nil {
		return fmt.Errorf("resolving build directory: %w", err)
	}

	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("creating build directory: %w", err)
	}

	if !opts.Keep {
		defer os.RemoveAll(buildDir)
	}

	moduleName := "ghoul-build/" + opts.BinaryName

	// Determine ghoul version and whether we need a local replace
	version, isDevel := ghoulModuleVersion()
	var localGhoulPath string
	if isDevel {
		localGhoulPath, err = findLocalGhoulRoot()
		if err != nil {
			return fmt.Errorf("running from source but %w — use `go install` for production use", err)
		}
		if opts.Verbose {
			fmt.Printf("  Using local ghoul module: %s\n", localGhoulPath)
		}
	}

	// Generate go.mod
	if err := generateGoMod(buildDir, moduleName, localGhoulPath); err != nil {
		return err
	}

	// go get ghoul dependency (skip if using local replace)
	if !isDevel {
		if err := goGet(buildDir, "github.com/archevel/ghoul@"+version); err != nil {
			return fmt.Errorf("getting ghoul dependency: %w", err)
		}
	}

	// go get third-party packages
	for _, entry := range entries {
		if isStdlib(entry.Package) {
			continue
		}
		if opts.Verbose {
			fmt.Printf("  go get %s\n", entry.Package)
		}
		if err := goGet(buildDir, entry.Package); err != nil {
			return fmt.Errorf("getting package %s: %w", entry.Package, err)
		}
	}

	// Generate mummies
	var succeeded []string
	var failed []string

	mummiesDir := filepath.Join(buildDir, "mummies")
	if err := os.MkdirAll(mummiesDir, 0755); err != nil {
		return fmt.Errorf("creating mummies directory: %w", err)
	}

	for _, entry := range entries {
		safe := safeName(entry.Package)
		outDir := filepath.Join(mummiesDir, safe+"_mummy")

		if opts.Verbose {
			fmt.Printf("  Mummifying %s... ", entry.Package)
		}

		mErr := embalmer.Mummify(&embalmer.MummificationConfig{
			ImportPath:      entry.Package,
			WorkDir:         buildDir,
			OutputDir:       outDir,
			Verbose:         false,
			SkipUnwrappable: entry.SkipUnwrappable,
		})
		if mErr != nil {
			if opts.Verbose {
				fmt.Printf("✗ %s\n", mErr)
			}
			failed = append(failed, entry.Package)
			os.RemoveAll(outDir)
			continue
		}
		if opts.Verbose {
			fmt.Println("✓")
		}
		succeeded = append(succeeded, safe)
	}

	fmt.Printf("Mummified %d packages, %d failed\n", len(succeeded), len(failed))
	if len(failed) > 0 {
		fmt.Printf("Failed: %s\n", strings.Join(failed, ", "))
	}

	// Write prelude if needed
	if !opts.NoPrelude {
		preludePath := filepath.Join(buildDir, "prelude.ghl")
		if err := os.WriteFile(preludePath, []byte(ghoul.PreludeSource()), 0644); err != nil {
			return fmt.Errorf("writing prelude: %w", err)
		}
	}

	// Generate sarcophagus.go
	sarcFile, err := os.Create(filepath.Join(buildDir, "sarcophagus.go"))
	if err != nil {
		return fmt.Errorf("creating sarcophagus.go: %w", err)
	}
	if err := renderSarcophagus(sarcFile, moduleName, succeeded); err != nil {
		sarcFile.Close()
		return fmt.Errorf("rendering sarcophagus.go: %w", err)
	}
	sarcFile.Close()

	// Generate main.go
	mainFile, err := os.Create(filepath.Join(buildDir, "main.go"))
	if err != nil {
		return fmt.Errorf("creating main.go: %w", err)
	}
	if err := renderMain(mainFile, !opts.NoPrelude); err != nil {
		mainFile.Close()
		return fmt.Errorf("rendering main.go: %w", err)
	}
	mainFile.Close()

	// Run go mod tidy
	if err := runCmd(buildDir, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	// Build the binary
	absBinary, err := filepath.Abs(opts.BinaryName)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	fmt.Printf("Building %s...\n", opts.BinaryName)
	if err := runCmd(buildDir, "go", "build", "-o", absBinary, "."); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	fmt.Printf("Built %s successfully\n", opts.BinaryName)
	return nil
}

func generateGoMod(buildDir, moduleName, ghoulModulePath string) error {
	content := fmt.Sprintf("module %s\n\ngo 1.25\n", moduleName)
	if ghoulModulePath != "" {
		content += fmt.Sprintf("\nreplace github.com/archevel/ghoul => %s\n", ghoulModulePath)
	}
	return os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte(content), 0644)
}

func goGet(dir, pkg string) error {
	return runCmd(dir, "go", "get", pkg)
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isStdlib checks whether a package is a Go standard library package.
// Uses a simple heuristic: stdlib packages don't contain a dot in the first path element.
func isStdlib(pkg string) bool {
	first := pkg
	if i := strings.Index(pkg, "/"); i >= 0 {
		first = pkg[:i]
	}
	return !strings.Contains(first, ".")
}
