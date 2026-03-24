package wraith

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuiltInTypeTemplateForInteger(t *testing.T) {
	tm, err := NewTypeMapper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	err = tm.GenerateArgumentConversion(ArgConversionInfo{
		N: 0, Name: "a", Type: "int", BuiltInType: "Integer",
	}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := buf.String()
	if !strings.Contains(code, "ghoulArg_a := args.First()") {
		t.Errorf("expected args.First() extraction, got:\n%s", code)
	}
	if !strings.Contains(code, "a_val, ok := ghoulArg_a.(e.Integer)") {
		t.Errorf("expected single type assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "a := int(a_val)") {
		t.Errorf("expected type cast, got:\n%s", code)
	}
	if !strings.Contains(code, "e.TypeName(ghoulArg_a)") {
		t.Errorf("expected TypeName in error message, got:\n%s", code)
	}
	if strings.Contains(code, "Foreign") {
		t.Errorf("should not reference Foreign, got:\n%s", code)
	}
	if strings.Contains(code, "Val()") {
		t.Errorf("should not call Val(), got:\n%s", code)
	}
}

func TestBuiltInTypeTemplateForString(t *testing.T) {
	tm, _ := NewTypeMapper()
	var buf bytes.Buffer
	tm.GenerateArgumentConversion(ArgConversionInfo{
		N: 0, Name: "s", Type: "string", BuiltInType: "String",
	}, &buf)

	code := buf.String()
	if !strings.Contains(code, "s_val, ok := ghoulArg_s.(e.String)") {
		t.Errorf("expected String assertion, got:\n%s", code)
	}
	if !strings.Contains(code, `expected string for parameter 's'`) {
		t.Errorf("expected human-readable error message, got:\n%s", code)
	}
}

func TestBuiltInTypeTemplateForBoolean(t *testing.T) {
	tm, _ := NewTypeMapper()
	var buf bytes.Buffer
	tm.GenerateArgumentConversion(ArgConversionInfo{
		N: 0, Name: "flag", Type: "bool", BuiltInType: "Boolean",
	}, &buf)

	code := buf.String()
	if !strings.Contains(code, "flag_val, ok := ghoulArg_flag.(e.Boolean)") {
		t.Errorf("expected Boolean assertion, got:\n%s", code)
	}
}

func TestForeignTypeTemplateUsesMummy(t *testing.T) {
	tm, _ := NewTypeMapper()
	var buf bytes.Buffer
	tm.GenerateArgumentConversion(ArgConversionInfo{
		N: 0, Name: "p", Type: "testpkg.Person", BuiltInType: "",
	}, &buf)

	code := buf.String()
	if !strings.Contains(code, "*mummy.Mummy") {
		t.Errorf("expected mummy.Mummy assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "mummy_p.Unwrap().(testpkg.Person)") {
		t.Errorf("expected Unwrap() call, got:\n%s", code)
	}
	if !strings.Contains(code, "mummy_p.Unwrap().(*testpkg.Person)") {
		t.Errorf("expected pointer fallback, got:\n%s", code)
	}
	if strings.Contains(code, "Foreign") {
		t.Errorf("should not reference Foreign, got:\n%s", code)
	}
}

func TestConvertValueToExpressionPrimitives(t *testing.T) {
	tm, _ := NewTypeMapper()

	cases := []struct {
		goType   string
		expected string
	}{
		{"int", "e.Integer(x)"},
		{"string", "e.String(x)"},
		{"bool", "e.Boolean(x)"},
		{"float64", "e.Float(x)"},
	}

	for _, c := range cases {
		result := tm.convertValueToExpression("x", c.goType)
		if result != c.expected {
			t.Errorf("convertValueToExpression(x, %s) = %s, expected %s", c.goType, result, c.expected)
		}
	}
}

func TestQualifiedTypeToAlias(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"int", "int"},
		{"string", "string"},
		{"github.com/archevel/ghoul/testpkg.Person", "testpkg.Person"},
		{"*github.com/archevel/ghoul/testpkg.Person", "*testpkg.Person"},
		{"**github.com/archevel/ghoul/testpkg.Person", "**testpkg.Person"},
		{"[]github.com/archevel/ghoul/testpkg.Person", "[]testpkg.Person"},
		{"fmt.Stringer", "fmt.Stringer"},
		{"Person", "Person"},
	}
	for _, c := range cases {
		result := qualifiedTypeToAlias(c.input)
		if result != c.expected {
			t.Errorf("qualifiedTypeToAlias(%q) = %q, expected %q", c.input, result, c.expected)
		}
	}
}

func TestFunctionTypeTemplateGeneratesAdapter(t *testing.T) {
	tm, _ := NewTypeMapper()
	var buf bytes.Buffer
	err := tm.GenerateArgumentConversion(ArgConversionInfo{
		N:    0,
		Name: "callback",
		Type: "func(int, int) int",
		FuncSignature: &FuncSignatureInfo{
			Params:  []FuncParamInfo{{Type: "int", GhoulType: "Integer"}, {Type: "int", GhoulType: "Integer"}},
			Results: []FuncParamInfo{{Type: "int", GhoulType: "Integer"}},
		},
	}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := buf.String()
	if !strings.Contains(code, "Function") {
		t.Errorf("expected Function type assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "e.Integer") {
		t.Errorf("expected Ghoul type conversion in adapter, got:\n%s", code)
	}
	if !strings.Contains(code, "mummy.Entomb") || !strings.Contains(code, "Unwrap") {
		// Should not use mummy for function params — it wraps a Ghoul Function
	}
}

func TestFunctionTypeTemplateVoidReturn(t *testing.T) {
	tm, _ := NewTypeMapper()
	var buf bytes.Buffer
	err := tm.GenerateArgumentConversion(ArgConversionInfo{
		N:    0,
		Name: "handler",
		Type: "func(int)",
		FuncSignature: &FuncSignatureInfo{
			Params:  []FuncParamInfo{{Type: "int", GhoulType: "Integer"}},
			Results: nil,
		},
	}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := buf.String()
	if !strings.Contains(code, "Function") {
		t.Errorf("expected Function type assertion, got:\n%s", code)
	}
}

func TestForeignTypeTemplateHandlesNil(t *testing.T) {
	tm, _ := NewTypeMapper()
	var buf bytes.Buffer
	tm.GenerateArgumentConversion(ArgConversionInfo{
		N: 0, Name: "handler", Type: "http.Handler", BuiltInType: "",
	}, &buf)

	code := buf.String()
	if !strings.Contains(code, "Unwrap() != nil") {
		t.Errorf("expected nil check in foreign template, got:\n%s", code)
	}
	if !strings.Contains(code, "var handler http.Handler") {
		t.Errorf("expected var declaration for nil case, got:\n%s", code)
	}
}

func TestConvertValueToExpressionComplexType(t *testing.T) {
	tm, _ := NewTypeMapper()
	result := tm.convertValueToExpression("result", "*testpkg.Person")
	if result != `mummy.Entomb(result, "*testpkg.Person")` {
		t.Errorf("expected mummy.Entomb, got: %s", result)
	}
}
