package wraith

import (
	"bytes"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSliceConstructorOutput(t *testing.T) {
	config := &Config{
		PackagePath: ".",
		OutputFile:  "/dev/null",
		PackageName: "test_sarcophagus",
	}
	g, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	structInfo := StructInfo{Name: "Person"}
	wrapper := g.generateSliceConstructor(structInfo, "github.com/example/pkg")

	if wrapper.GhoulName != "person-slice" {
		t.Errorf("expected ghoul name 'person-slice', got '%s'", wrapper.GhoulName)
	}
	if wrapper.GoFuncName != "mummy_personslice" {
		t.Errorf("expected go func name 'w_personslice', got '%s'", wrapper.GoFuncName)
	}
	if !strings.Contains(wrapper.GeneratedCode, "[]*pkg.Person") {
		t.Errorf("expected []*pkg.Person in generated code, got:\n%s", wrapper.GeneratedCode)
	}
	if !strings.Contains(wrapper.GeneratedCode, "MummyNodeVal") {
		t.Errorf("expected MummyNodeVal in result, got:\n%s", wrapper.GeneratedCode)
	}
	if !strings.Contains(wrapper.GeneratedCode, "_e.NodeTypeName") {
		t.Errorf("expected _e.NodeTypeName in error message, got:\n%s", wrapper.GeneratedCode)
	}
}

func TestGenerateInterfaceMethodWrapperOutput(t *testing.T) {
	config := &Config{
		PackagePath: ".",
		OutputFile:  "/dev/null",
		PackageName: "test_sarcophagus",
	}
	g, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	method := FunctionInfo{
		Name: "Write",
		Params: []ParameterInfo{
			{Name: "data", Type: fakeType("[]byte")},
		},
		Results: []ParameterInfo{
			{Name: "n", Type: fakeType("int")},
			{Name: "err", Type: fakeType("error")},
		},
	}

	wrapper, err := g.generateInterfaceMethodWrapper("Writer", method, "github.com/example/pkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wrapper.GhoulName != "writer-write" {
		t.Errorf("expected 'writer-write', got '%s'", wrapper.GhoulName)
	}
	if wrapper.GoFuncName != "mummy_writerwrite" {
		t.Errorf("expected 'writerwrite', got '%s'", wrapper.GoFuncName)
	}
	if !strings.Contains(wrapper.GeneratedCode, "pkg.Writer") {
		t.Errorf("expected interface type assertion, got:\n%s", wrapper.GeneratedCode)
	}
	// Interface receiver should NOT have pointer fallback
	if strings.Contains(wrapper.GeneratedCode, "Unwrap().(*pkg.Writer)") {
		t.Errorf("interface unwrap should not have pointer fallback, got:\n%s", wrapper.GeneratedCode)
	}
}

func TestGenerateInterfaceMethodWrapperNaming(t *testing.T) {
	config := &Config{
		PackagePath: ".",
		OutputFile:  "/dev/null",
		PackageName: "test_sarcophagus",
	}
	g, _ := NewGenerator(config)

	method := FunctionInfo{
		Name:    "DoStuff",
		Params:  nil,
		Results: nil,
	}

	wrapper, _ := g.generateInterfaceMethodWrapper("MyInterface", method, "example/pkg")
	if wrapper.GhoulName != "myinterface-dostuff" {
		t.Errorf("expected 'myinterface-dostuff', got '%s'", wrapper.GhoulName)
	}
}

func TestMethodNamingConvention(t *testing.T) {
	config := &Config{
		PackagePath: ".",
		OutputFile:  "/dev/null",
		PackageName: "test_sarcophagus",
	}
	g, _ := NewGenerator(config)

	funcInfo := FunctionInfo{
		Name:   "GetAge",
		Params: nil,
		Results: []ParameterInfo{
			{Name: "age", Type: fakeType("int")},
		},
		Receiver: &ParameterInfo{
			Name: "p",
			Type: fakeType("*example.Person"),
		},
	}

	wrapper, err := g.processFunctionInfo(funcInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wrapper.GhoulName != "person-get-age" {
		t.Errorf("expected 'person-get-age', got '%s'", wrapper.GhoulName)
	}
	if wrapper.GoFuncName != "mummy_persongetage" {
		t.Errorf("expected 'w_persongetage', got '%s'", wrapper.GoFuncName)
	}
}

func TestConstructorNaming(t *testing.T) {
	config := &Config{
		PackagePath: ".",
		OutputFile:  "/dev/null",
		PackageName: "test_sarcophagus",
	}
	g, _ := NewGenerator(config)

	structInfo := StructInfo{Name: "Widget"}
	wrapper := g.generateSliceConstructor(structInfo, "example/pkg")

	if wrapper.GhoulName != "widget-slice" {
		t.Errorf("expected 'widget-slice', got '%s'", wrapper.GhoulName)
	}
}

func TestGenerateResultHandlingVoid(t *testing.T) {
	config := &Config{PackagePath: ".", OutputFile: "/dev/null", PackageName: "test"}
	g, _ := NewGenerator(config)
	var buf bytes.Buffer
	g.generateResultHandling(nil, &buf)
	if !strings.Contains(buf.String(), "_e.Nil") {
		t.Errorf("expected _e.Nil for void result, got:\n%s", buf.String())
	}
}

func TestGenerateResultHandlingSingleReturn(t *testing.T) {
	config := &Config{PackagePath: ".", OutputFile: "/dev/null", PackageName: "test"}
	g, _ := NewGenerator(config)
	var buf bytes.Buffer
	g.generateResultHandling([]ResultConversionInfo{
		{Index: 0, Type: "int", Name: "result0"},
	}, &buf)
	if !strings.Contains(buf.String(), "_e.IntNode(int64(result0))") {
		t.Errorf("expected IntNode conversion, got:\n%s", buf.String())
	}
}

func TestGenerateResultHandlingWithError(t *testing.T) {
	config := &Config{PackagePath: ".", OutputFile: "/dev/null", PackageName: "test"}
	g, _ := NewGenerator(config)
	var buf bytes.Buffer
	g.generateResultHandling([]ResultConversionInfo{
		{Index: 0, Type: "int", Name: "result0"},
		{Index: 1, Type: "error", Name: "err"},
	}, &buf)
	code := buf.String()
	if !strings.Contains(code, "err != nil") {
		t.Errorf("expected error check, got:\n%s", code)
	}
	if !strings.Contains(code, "_e.IntNode(int64(result0))") {
		t.Errorf("expected IntNode conversion, got:\n%s", code)
	}
}

func TestGenerateResultHandlingMultipleReturns(t *testing.T) {
	config := &Config{PackagePath: ".", OutputFile: "/dev/null", PackageName: "test"}
	g, _ := NewGenerator(config)
	var buf bytes.Buffer
	g.generateResultHandling([]ResultConversionInfo{
		{Index: 0, Type: "int", Name: "a"},
		{Index: 1, Type: "string", Name: "b"},
	}, &buf)
	code := buf.String()
	if !strings.Contains(code, "_e.NewListNode") {
		t.Errorf("expected NewListNode for multi-return, got:\n%s", code)
	}
}

func TestGetPackagePrefix(t *testing.T) {
	cases := []struct {
		input, expected string
	}{
		{"github.com/foo/bar", "bar"},
		{"fmt", "fmt"},
		{"github.com/archevel/ghoul/testpkg", "testpkg"},
		{"", ""},
	}
	for _, c := range cases {
		result := getPackagePrefix(c.input)
		if result != c.expected {
			t.Errorf("getPackagePrefix(%q) = %q, expected %q", c.input, result, c.expected)
		}
	}
}

func TestGeneratedCodeContainsMultipleReturnTypes(t *testing.T) {
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

	// Divide returns (int, error) — should have both error handling and IntNode conversion
	if !strings.Contains(code, "function failed") {
		t.Error("expected error handling for Divide")
	}

	// Variadic functions should be wrapped
	if !strings.Contains(code, "nums...") {
		t.Error("expected variadic spread for Variadic")
	}

	// JoinWith has mixed regular + variadic params
	if !strings.Contains(code, "parts...") {
		t.Error("expected variadic spread for JoinWith")
	}
}

func TestPossessPackageCreatesSarcophagus(t *testing.T) {
	// Use the testpkg as the target
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
		t.Fatalf("possession failed: %v", err)
	}

	outputFile := filepath.Join(outputDir, "testpkg.go")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	code := string(content)

	// Verify key elements are present
	checks := []struct {
		desc    string
		content string
	}{
		{"package declaration", "package testpkg_sarcophagus"},
		{"mummy import", "github.com/archevel/ghoul/mummy"},
		{"testpkg import", "github.com/archevel/ghoul/testpkg"},
		{"constructor", "make-person"},
		{"slice constructor", "person-slice"},
		{"interface method", "greeter-greet"},
		{"method naming", "person-get-age"},
		{"callback adapter", "FuncVal"},
		{"nil handling", "ForeignVal != nil"},
		{"RegisterFunctions", "func RegisterFunctions"},
		{"init registration", "_mummy.RegisterSarcophagus"},
		{"registerWithPrefix", "func registerWithPrefix"},
		{"RegisterIfAllowed", "_mummy.RegisterIfAllowed"},
		{"node signature", "[]*_e.Node"},
		{"node return", "*_e.Node"},
	}

	for _, c := range checks {
		if !strings.Contains(code, c.content) {
			t.Errorf("expected %s (%q) in generated code", c.desc, c.content)
		}
	}
}

func TestFunctionsUsingExternalTypesAreWrapped(t *testing.T) {
	code := possessAndRead(t)

	// ReadAll uses io.Reader — the function should be wrapped and
	// the generated code should import "io"
	if !strings.Contains(code, "readall") {
		t.Fatal("ReadAll should be wrapped — io.Reader is a standard external type")
	}
	if !strings.Contains(code, `"io"`) {
		t.Error("generated code uses io.Reader but 'io' is not in imports")
	}
}

func TestFunctionsReturningExternalTypesAreWrapped(t *testing.T) {
	code := possessAndRead(t)

	// OpenFile returns *os.File — should be wrapped with mummy
	if !strings.Contains(code, "openfile") {
		t.Fatal("OpenFile should be wrapped — *os.File is a standard return type")
	}
	if !strings.Contains(code, `"os"`) {
		t.Error("generated code returns *os.File but 'os' is not in imports")
	}
}

func TestUnexportedReturnTypesSkipped(t *testing.T) {
	code := possessAndRead(t)

	// MakeResult returns *result (unexported) — should be skipped
	if strings.Contains(code, "makeresult") {
		t.Error("MakeResult uses unexported return type — should be skipped")
	}

	// GetResultValue takes *result (unexported) — should be skipped
	if strings.Contains(code, "getresultvalue") {
		t.Error("GetResultValue uses unexported param type — should be skipped")
	}
}

func TestCamelCaseToKebabNaming(t *testing.T) {
	code := possessAndRead(t)

	if !strings.Contains(code, `"is-even"`) {
		t.Error("expected ghoul name 'is-even' for IsEven (CamelCase → kebab-case)")
	}
	if !strings.Contains(code, `"split-name-age"`) || !strings.Contains(code, `"sum-floats"`) {
		t.Log("CamelCase to kebab-case conversion not splitting on word boundaries")
	}
}

func TestMultiReturnFunctionHandled(t *testing.T) {
	code := possessAndRead(t)

	// SplitNameAge returns (string, int) — should be wrapped with list return
	if strings.Contains(code, "splitnameage") {
		if !strings.Contains(code, "_e.NewListNode") {
			t.Error("multi-return SplitNameAge should use NewListNode")
		}
	}
}

func TestDocCommentsAreSingleLine(t *testing.T) {
	code := possessAndRead(t)

	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.HasPrefix(trimmed, "Special cases") || strings.Contains(line, "±") {
			t.Errorf("multi-line doc content leaked into code: %s", line)
		}
	}
}

func possessAndRead(t *testing.T) string {
	t.Helper()

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
		t.Fatalf("PossessPackage failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "testpkg.go"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	return string(content)
}

// fakeType implements types.Type for testing without real Go packages
type fakeType string

func (f fakeType) Underlying() types.Type { return f }
func (f fakeType) String() string         { return string(f) }
