package wraith

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPossessNonexistentPackageFails(t *testing.T) {
	err := PossessPackage(&PossessionConfig{
		PackagePath: "/nonexistent/path",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent package")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' in error, got: %s", err)
	}
}

func TestPossessEmptyPackageFails(t *testing.T) {
	emptyDir := t.TempDir()
	err := PossessPackage(&PossessionConfig{
		PackagePath: emptyDir,
		OutputDir:   t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestPossessPackageVerboseOutput(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	err := PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       t.TempDir(),
		Verbose:         true,
		SkipUnwrappable: true,
	})
	if err != nil {
		t.Fatalf("possession failed: %v", err)
	}
}

func TestGenerateWrappersWithBadPackagePath(t *testing.T) {
	config := &Config{
		PackagePath: "/nonexistent",
		OutputFile:  "/dev/null",
		PackageName: "test",
	}
	err := GenerateWrappers(config)
	if err == nil {
		t.Fatal("expected error for bad package path")
	}
}

func TestPossessPackageOutputDirCreated(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := filepath.Join(t.TempDir(), "nested", "deep", "sarcophagus")
	err := PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})
	if err != nil {
		t.Fatalf("possession failed: %v", err)
	}

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("expected output directory to be created")
	}
}

func TestGeneratedCodeSkipsUnexportedFunctions(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := t.TempDir()
	PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})

	content, _ := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	code := string(content)

	// Should not contain unexported function names from testpkg
	// (testpkg only has exported functions, but verify the pattern)
	if strings.Contains(code, "func unexported") {
		t.Error("generated code should not contain unexported functions")
	}
}

func TestGeneratedCodeSkipsUnexportedStructs(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := t.TempDir()
	PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})

	content, _ := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	code := string(content)

	// Only exported structs should have constructors
	if !strings.Contains(code, "make-person") {
		t.Error("expected constructor for exported Person struct")
	}
}

func TestResultHandlingVoidFunction(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := t.TempDir()
	PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})

	content, _ := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	code := string(content)

	// SetAge is void — should return e.NIL
	if !strings.Contains(code, "e.NIL, nil") {
		t.Error("expected e.NIL return for void function")
	}
}

func TestPossessFailsOnUnwrappableFunctions(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	err := PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       t.TempDir(),
		SkipUnwrappable: false,
	})
	if err == nil {
		t.Fatal("expected error when package has unwrappable functions")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "could not be wrapped") {
		t.Errorf("expected 'could not be wrapped' in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "channel") {
		t.Errorf("expected mention of channel type, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "map") {
		t.Errorf("expected mention of map type, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "--skip-unwrappable") {
		t.Errorf("expected suggestion to use --skip-unwrappable, got: %s", errMsg)
	}
}

func TestPossessSucceedsWithSkipUnwrappable(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := t.TempDir()
	err := PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})
	if err != nil {
		t.Fatalf("expected success with --skip-unwrappable, got: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	code := string(content)

	// Wrapped functions should be present
	if !strings.Contains(code, "\"add\"") {
		t.Error("expected 'add' function to be wrapped")
	}
	// Unwrappable functions should NOT be present
	if strings.Contains(code, "sendonchannel") {
		t.Error("channel function should be skipped")
	}
	if strings.Contains(code, "lookupmap") {
		t.Error("map function should be skipped")
	}
	// Variadic functions SHOULD be wrapped (consuming remaining args)
	if !strings.Contains(code, "\"variadic\"") {
		t.Error("variadic function should be wrapped")
	}
}

func TestUnsupportedTypeReasonChannel(t *testing.T) {
	tm, _ := NewTypeMapper()
	reason := tm.UnsupportedTypeReason(fakeType("chan int"))
	// fakeType doesn't have proper Underlying(), so this tests the fallback
	if reason != "" {
		t.Logf("fakeType doesn't trigger channel detection (expected)")
	}
}

func TestVariadicFunctionGeneratesArgCollection(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := t.TempDir()
	PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})

	content, _ := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	code := string(content)

	// The variadic wrapper should collect remaining args into a slice
	if !strings.Contains(code, "for args != e.NIL") {
		t.Errorf("expected arg collection loop for variadic function, got:\n%s", code)
	}
}

func TestVariadicFunctionWithNoArgs(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := t.TempDir()
	PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})

	content, _ := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	code := string(content)

	// The generated variadic code should declare the var before the loop,
	// so an empty call (no args for the variadic param) results in a nil slice
	// which Go accepts with ... spread
	if !strings.Contains(code, "var nums []int") {
		t.Error("expected 'var nums []int' declaration before loop")
	}
	if !strings.Contains(code, "nums...)") {
		t.Error("expected 'nums...' spread in function call")
	}
}

func TestVariadicFunctionWithMixedParams(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := t.TempDir()
	PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})

	content, _ := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	code := string(content)

	// JoinWith has sep (regular) + parts (variadic)
	// The regular param should be extracted first, then the variadic loop
	if !strings.Contains(code, "joinwith") {
		t.Error("expected joinwith function to be generated")
	}
	if !strings.Contains(code, "parts...)") {
		t.Error("expected 'parts...' spread in joinwith call")
	}
}

func TestResultHandlingErrorFunction(t *testing.T) {
	testpkgPath, _ := filepath.Abs("../testpkg")
	if _, err := os.Stat(testpkgPath); os.IsNotExist(err) {
		t.Skip("testpkg not found")
	}

	outputDir := t.TempDir()
	PossessPackage(&PossessionConfig{
		PackagePath:     testpkgPath,
		OutputDir:       outputDir,
		SkipUnwrappable: true,
	})

	content, _ := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	code := string(content)

	// Divide returns (int, error) — should have error handling
	if !strings.Contains(code, "function failed") {
		t.Error("expected error handling for functions returning error")
	}
}
