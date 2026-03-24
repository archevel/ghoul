package wraith

import (
	"bytes"
	"go/types"
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
		{"map[string]int", "map[string]int"},
		{"map[github.com/pkg.Foo]github.com/pkg.Bar", "map[pkg.Foo]pkg.Bar"},
		{"chan int", "chan int"},
		{"chan<- int", "chan<- int"},
		{"<-chan int", "<-chan int"},
		{"chan github.com/pkg.Foo", "chan pkg.Foo"},
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

func TestGhoulToGoConversionAllTypes(t *testing.T) {
	tm, _ := NewTypeMapper()
	cases := []struct {
		param    FuncParamInfo
		expected string
	}{
		{FuncParamInfo{Type: "int", GhoulType: "Integer"}, "int(x.(e.Integer))"},
		{FuncParamInfo{Type: "string", GhoulType: "String"}, "string(x.(e.String))"},
		{FuncParamInfo{Type: "bool", GhoulType: "Boolean"}, "bool(x.(e.Boolean))"},
		{FuncParamInfo{Type: "float64", GhoulType: "Float"}, "float64(x.(e.Float))"},
		{FuncParamInfo{Type: "pkg.Foo", GhoulType: ""}, "x.(*mummy.Mummy).Unwrap().(pkg.Foo)"},
	}
	for _, c := range cases {
		result := tm.ghoulToGoConversion("x", c.param)
		if result != c.expected {
			t.Errorf("ghoulToGoConversion(x, %v) = %q, expected %q", c.param, result, c.expected)
		}
	}
}

func TestVariadicConversionForPrimitiveType(t *testing.T) {
	tm, _ := NewTypeMapper()
	var buf bytes.Buffer
	err := tm.GenerateArgumentConversion(ArgConversionInfo{
		Name:        "nums",
		Type:        "[]int",
		BuiltInType: "Integer",
		IsVariadic:  true,
	}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	code := buf.String()
	if !strings.Contains(code, "var nums []int") {
		t.Errorf("expected var declaration, got:\n%s", code)
	}
	if !strings.Contains(code, "for args != e.NIL") {
		t.Errorf("expected loop, got:\n%s", code)
	}
	if !strings.Contains(code, "e.Integer") {
		t.Errorf("expected Integer assertion, got:\n%s", code)
	}
}

func TestVariadicConversionForForeignType(t *testing.T) {
	tm, _ := NewTypeMapper()
	var buf bytes.Buffer
	err := tm.GenerateArgumentConversion(ArgConversionInfo{
		Name:       "items",
		Type:       "[]pkg.Item",
		IsVariadic: true,
	}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	code := buf.String()
	if !strings.Contains(code, "*mummy.Mummy") {
		t.Errorf("expected mummy assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "Unwrap().(pkg.Item)") {
		t.Errorf("expected Unwrap with type, got:\n%s", code)
	}
}

func TestChannelTypeIsSupported(t *testing.T) {
	tm, _ := NewTypeMapper()
	chanType := types.NewChan(types.SendRecv, types.Typ[types.Int])
	reason := tm.UnsupportedTypeReason(chanType)
	if reason != "" {
		t.Errorf("channels should be supported, got reason: %s", reason)
	}
}

func TestMapTypeIsSupported(t *testing.T) {
	tm, _ := NewTypeMapper()
	mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	reason := tm.UnsupportedTypeReason(mapType)
	if reason != "" {
		t.Errorf("maps should be supported, got reason: %s", reason)
	}
}

func TestUnsupportedTypeReasonForSupportedType(t *testing.T) {
	tm, _ := NewTypeMapper()
	reason := tm.UnsupportedTypeReason(types.Typ[types.Int])
	if reason != "" {
		t.Errorf("expected empty reason for int, got: %s", reason)
	}
}

func TestUnsupportedTypeReasonForNil(t *testing.T) {
	tm, _ := NewTypeMapper()
	reason := tm.UnsupportedTypeReason(nil)
	if reason != "" {
		t.Errorf("expected empty reason for nil, got: %s", reason)
	}
}

func TestFindMatchingBracketSuccess(t *testing.T) {
	result := findMatchingBracket("map[string]int", 3)
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

func TestFindMatchingBracketNested(t *testing.T) {
	result := findMatchingBracket("map[map[int]int]string", 3)
	if result != 15 {
		t.Errorf("expected 15, got %d", result)
	}
}

func TestFindMatchingBracketNoMatch(t *testing.T) {
	result := findMatchingBracket("map[string", 3)
	if result != -1 {
		t.Errorf("expected -1 for no match, got %d", result)
	}
}

func TestIsErrorTypeWithNil(t *testing.T) {
	if isErrorType(nil) {
		t.Error("nil should not be an error type")
	}
}

func TestIsErrorTypeWithError(t *testing.T) {
	// Create a real error type via go/types universe
	errorType := types.Universe.Lookup("error").Type()
	if !isErrorType(errorType) {
		t.Error("error type should be recognized")
	}
}

func TestIsErrorTypeWithNonError(t *testing.T) {
	if isErrorType(types.Typ[types.Int]) {
		t.Error("int should not be an error type")
	}
}

func TestQualifiedTypeToAliasMapNoClosingBracket(t *testing.T) {
	// Malformed map type — should handle gracefully
	result := qualifiedTypeToAlias("map[string")
	// Falls through to the default path since findMatchingBracket returns -1
	if result != "map[string" {
		t.Errorf("expected 'map[string', got '%s'", result)
	}
}

func TestConvertValueToExpressionComplexType(t *testing.T) {
	tm, _ := NewTypeMapper()
	result := tm.convertValueToExpression("result", "*testpkg.Person")
	if result != `mummy.Entomb(result, "*testpkg.Person")` {
		t.Errorf("expected mummy.Entomb, got: %s", result)
	}
}
