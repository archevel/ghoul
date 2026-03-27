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
	if wrapper.GoFuncName != "personslice" {
		t.Errorf("expected go func name 'personslice', got '%s'", wrapper.GoFuncName)
	}
	if !strings.Contains(wrapper.GeneratedCode, "[]*pkg.Person") {
		t.Errorf("expected []*pkg.Person in generated code, got:\n%s", wrapper.GeneratedCode)
	}
	if !strings.Contains(wrapper.GeneratedCode, "MummyNodeVal") {
		t.Errorf("expected MummyNodeVal in result, got:\n%s", wrapper.GeneratedCode)
	}
	if !strings.Contains(wrapper.GeneratedCode, "e.NodeTypeName") {
		t.Errorf("expected e.NodeTypeName in error message, got:\n%s", wrapper.GeneratedCode)
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
	if wrapper.GoFuncName != "writerwrite" {
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
	if wrapper.GhoulName != "person-getage" {
		t.Errorf("expected 'person-getage', got '%s'", wrapper.GhoulName)
	}
	if wrapper.GoFuncName != "persongetage" {
		t.Errorf("expected 'persongetage', got '%s'", wrapper.GoFuncName)
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
	if !strings.Contains(buf.String(), "e.Nil") {
		t.Errorf("expected e.Nil for void result, got:\n%s", buf.String())
	}
}

func TestGenerateResultHandlingSingleReturn(t *testing.T) {
	config := &Config{PackagePath: ".", OutputFile: "/dev/null", PackageName: "test"}
	g, _ := NewGenerator(config)
	var buf bytes.Buffer
	g.generateResultHandling([]ResultConversionInfo{
		{Index: 0, Type: "int", Name: "result0"},
	}, &buf)
	if !strings.Contains(buf.String(), "e.IntNode(int64(result0))") {
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
	if !strings.Contains(code, "e.IntNode(int64(result0))") {
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
	if !strings.Contains(code, "e.NewListNode") {
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
		{"method naming", "person-getage"},
		{"callback adapter", "FuncVal"},
		{"nil handling", "ForeignVal != nil"},
		{"RegisterFunctions", "func RegisterFunctions"},
		{"init registration", "mummy.RegisterSarcophagus"},
		{"registerWithPrefix", "func registerWithPrefix"},
		{"RegisterIfAllowed", "mummy.RegisterIfAllowed"},
		{"node signature", "[]*e.Node"},
		{"node return", "*e.Node"},
	}

	for _, c := range checks {
		if !strings.Contains(code, c.content) {
			t.Errorf("expected %s (%q) in generated code", c.desc, c.content)
		}
	}
}

// fakeType implements types.Type for testing without real Go packages
type fakeType string

func (f fakeType) Underlying() types.Type { return f }
func (f fakeType) String() string         { return string(f) }
