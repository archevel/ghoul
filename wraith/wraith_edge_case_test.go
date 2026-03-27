package wraith

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

// --- TypeMapper edge cases ---

func TestTypeMapperPrimitiveMapHasExpectedEntries(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		goType   string
		expected string
	}{
		{"bool", "Boolean"},
		{"string", "String"},
		{"int", "Integer"},
		{"int64", "Integer"},
		{"float64", "Float"},
		{"float32", "Float"},
		{"uint", "Integer"},
	}

	for _, c := range cases {
		if tm.primitiveMap[c.goType] != c.expected {
			t.Errorf("expected %s → %s, got %s", c.goType, c.expected, tm.primitiveMap[c.goType])
		}
	}
}

func TestTypeMapperNonPrimitive(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tm.primitiveMap["custom.Type"]; ok {
		t.Error("custom type should not be primitive")
	}
}

// --- GenerateArgumentConversion edge cases ---

func TestGenerateArgumentConversionBuiltIn(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	info := ArgConversionInfo{
		Name:        "x",
		Type:        "int64",
		BuiltInType: "Integer",
	}
	err = tm.GenerateArgumentConversion(info, &buf)
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "args[argIdx]") {
		t.Errorf("expected args[argIdx] in output, got:\n%s", output)
	}
	if !strings.Contains(output, "IntegerNode") {
		t.Errorf("expected IntegerNode check in output, got:\n%s", output)
	}
}

func TestGenerateArgumentConversionForeign(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	info := ArgConversionInfo{
		Name: "p",
		Type: "pkg.Person",
	}
	err = tm.GenerateArgumentConversion(info, &buf)
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "MummyNode") {
		t.Errorf("expected MummyNode check in output, got:\n%s", output)
	}
	if !strings.Contains(output, "ForeignVal") {
		t.Errorf("expected ForeignVal in output, got:\n%s", output)
	}
}

func TestGenerateArgumentConversionMissingTemplate(t *testing.T) {
	tm := &TypeMapper{
		primitiveMap: map[string]string{},
		templates:    make(map[string]*template.Template),
	}
	var buf bytes.Buffer
	info := ArgConversionInfo{
		Name:        "x",
		Type:        "int64",
		BuiltInType: "Integer",
	}
	err := tm.GenerateArgumentConversion(info, &buf)
	if err == nil {
		t.Error("expected error for missing template")
	}
}

func TestConvertValueToExpressionBoolean(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.convertValueToExpression("myBool", "bool")
	if !strings.Contains(result, "BoolNode") {
		t.Errorf("expected BoolNode, got %s", result)
	}
}

func TestConvertValueToExpressionInteger(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.convertValueToExpression("myInt", "int")
	if !strings.Contains(result, "IntNode") {
		t.Errorf("expected IntNode, got %s", result)
	}
}

func TestConvertValueToExpressionFloat(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.convertValueToExpression("myFloat", "float64")
	if !strings.Contains(result, "FloatNode") {
		t.Errorf("expected FloatNode, got %s", result)
	}
}

func TestConvertValueToExpressionString(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.convertValueToExpression("myStr", "string")
	if !strings.Contains(result, "StrNode") {
		t.Errorf("expected StrNode, got %s", result)
	}
}

func TestConvertValueToExpressionForeign(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.convertValueToExpression("myVal", "pkg.Foo")
	if !strings.Contains(result, "MummyNodeVal") {
		t.Errorf("expected MummyNodeVal, got %s", result)
	}
}

func TestGhoulToGoConversionInteger(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.ghoulToGoConversion("expr", FuncParamInfo{Type: "int64", GhoulType: "Integer"})
	if !strings.Contains(result, "IntVal") {
		t.Errorf("expected IntVal, got %s", result)
	}
}

func TestGhoulToGoConversionString(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.ghoulToGoConversion("expr", FuncParamInfo{Type: "string", GhoulType: "String"})
	if !strings.Contains(result, "StrVal") {
		t.Errorf("expected StrVal, got %s", result)
	}
}

func TestGhoulToGoConversionBoolean(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.ghoulToGoConversion("expr", FuncParamInfo{Type: "bool", GhoulType: "Boolean"})
	if !strings.Contains(result, "BoolVal") {
		t.Errorf("expected BoolVal, got %s", result)
	}
}

func TestGhoulToGoConversionFloat(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.ghoulToGoConversion("expr", FuncParamInfo{Type: "float64", GhoulType: "Float"})
	if !strings.Contains(result, "FloatVal") {
		t.Errorf("expected FloatVal, got %s", result)
	}
}

func TestGhoulToGoConversionForeign(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatal(err)
	}
	result := tm.ghoulToGoConversion("expr", FuncParamInfo{Type: "pkg.Foo"})
	if !strings.Contains(result, "ForeignVal") {
		t.Errorf("expected ForeignVal, got %s", result)
	}
}

// --- getPackagePrefix ---

func TestGetPackagePrefixSimple(t *testing.T) {
	result := getPackagePrefix("github.com/foo/bar")
	if result != "bar" {
		t.Errorf("expected bar, got %s", result)
	}
}

func TestGetPackagePrefixSingleSegment(t *testing.T) {
	result := getPackagePrefix("mypkg")
	if result != "mypkg" {
		t.Errorf("expected mypkg, got %s", result)
	}
}
