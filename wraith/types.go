package wraith

import (
	"fmt"
	"go/types"
	"io"
	"text/template"
)

// TypeMapper handles conversion between Go types and Ghoul expressions
type TypeMapper struct {
	// Maps Go type names to Ghoul expression types
	primitiveMap map[string]string
	templates    map[string]*template.Template
}

// NewTypeMapper creates a new type mapper with standard conversions
func NewTypeMapper() (*TypeMapper, error) {
	tm := &TypeMapper{
		primitiveMap: map[string]string{
			"bool":    "Boolean",
			"string":  "String",
			"int":     "Integer",
			"int8":    "Integer",
			"int16":   "Integer",
			"int32":   "Integer",
			"int64":   "Integer",
			"uint":    "Integer",
			"uint8":   "Integer",
			"uint16":  "Integer",
			"uint32":  "Integer",
			"uint64":  "Integer",
			"float32": "Float",
			"float64": "Float",
			// TODO: Add slice support for primitive types
		},
		templates: make(map[string]*template.Template),
	}

	err := tm.initializeTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize type mapping templates: %w", err)
	}

	return tm, nil
}

// ArgConversionInfo holds information about converting a function argument
type ArgConversionInfo struct {
	N           int    // Argument index
	Name        string // Parameter name
	Type        string // Go type string
	BuiltInType string // Ghoul expression type (empty for Foreign)
}

// ResultConversionInfo holds information about converting function results
type ResultConversionInfo struct {
	Index int    // Result index
	Type  string // Go type string
	Name  string // Result variable name
}

// initializeTemplates creates the code generation templates
func (tm *TypeMapper) initializeTemplates() error {
	// Template for converting built-in types (int, string, bool, float)
	builtInTypeTemplate := `
	var {{.Name}} {{.Type}}

	switch v := args.First().(type) {
		case e.{{.BuiltInType}}:
			{{.Name}} = {{.Type}}(v)
		case *e.{{.BuiltInType}}:
			{{.Name}} = {{.Type}}(*v)
		case e.Foreign:
			switch vv := v.Val().(type) {
			case *{{.Type}}:
				{{.Name}} = *vv
			case {{.Type}}:
				{{.Name}} = vv
			default:
				return nil, fmt.Errorf("{{.Name}}: cannot convert argument to {{.Type}}, got %T", v.Val())
			}
		case *e.Foreign:
			switch vv := v.Val().(type) {
			case *{{.Type}}:
				{{.Name}} = *vv
			case {{.Type}}:
				{{.Name}} = vv
			default:
				return nil, fmt.Errorf("{{.Name}}: cannot convert argument to {{.Type}}, got %T", v.Val())
			}
		default:
			return nil, fmt.Errorf("{{.Name}}: expected {{.BuiltInType}} or Foreign, got %T", v)
	}

	// Move to next argument
	args, _ = args.Tail()
`

	// Template for converting structs and interfaces to Foreign
	foreignTypeTemplate := `
	var {{.Name}} {{.Type}}

	switch f := args.First().(type) {
	case e.Foreign:
		switch v := f.Val().(type) {
		case *{{.Type}}:
			{{.Name}} = *v
		case {{.Type}}:
			{{.Name}} = v
		default:
			return nil, fmt.Errorf("{{.Name}}: cannot convert argument to {{.Type}}, got %T", v)
		}
	case *e.Foreign:
		switch v := f.Val().(type) {
		case *{{.Type}}:
			{{.Name}} = *v
		case {{.Type}}:
			{{.Name}} = v
		default:
			return nil, fmt.Errorf("{{.Name}}: cannot convert argument to {{.Type}}, got %T", v)
		}
	default:
		return nil, fmt.Errorf("{{.Name}}: expected Foreign type containing {{.Type}}, got %T", f)
	}

	// Move to next argument
	args, _ = args.Tail()
`

	var err error
	tm.templates["builtin"], err = template.New("builtin").Parse(builtInTypeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse builtin type template: %w", err)
	}

	tm.templates["foreign"], err = template.New("foreign").Parse(foreignTypeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse foreign type template: %w", err)
	}

	return nil
}

// GenerateArgumentConversion generates code to convert a Ghoul argument to a Go type
func (tm *TypeMapper) GenerateArgumentConversion(info ArgConversionInfo, w io.Writer) error {
	var templateName string
	if info.BuiltInType != "" {
		templateName = "builtin"
	} else {
		templateName = "foreign"
	}

	template, exists := tm.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	return template.Execute(w, info)
}

// MapGoTypeToGhoul maps a Go type to its corresponding Ghoul expression type
func (tm *TypeMapper) MapGoTypeToGhoul(goType types.Type) (ghouType string, isForeign bool) {
	typeStr := goType.String()

	// Check for primitive types
	if ghoulType, exists := tm.primitiveMap[typeStr]; exists {
		return ghoulType, false
	}

	// Handle pointer types
	if ptr, ok := goType.(*types.Pointer); ok {
		if ghoulType, exists := tm.primitiveMap[ptr.Elem().String()]; exists {
			return ghoulType, false
		}
	}

	// Handle slices of primitive types
	if slice, ok := goType.(*types.Slice); ok {
		if tm.isPrimitiveType(slice.Elem()) {
			// For now, treat slices as Foreign
			// TODO: Implement List conversion for primitive slices
			return "", true
		}
	}

	// Default to Foreign for complex types
	return "", true
}

// isPrimitiveType checks if a type is a Go primitive that maps to Ghoul
func (tm *TypeMapper) isPrimitiveType(t types.Type) bool {
	_, exists := tm.primitiveMap[t.String()]
	return exists
}

// GenerateResultConversion generates code to convert Go results back to Ghoul expressions
func (tm *TypeMapper) GenerateResultConversion(results []ResultConversionInfo, funcName string, w io.Writer) error {
	// Generate function call
	fmt.Fprintf(w, "\n\t// Call original Go function\n")
	fmt.Fprintf(w, "\t")

	// Generate result assignment
	if len(results) > 0 {
		for i, result := range results {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, "%s", result.Name)
		}
		fmt.Fprintf(w, " := ")
	}

	fmt.Fprintf(w, "%s(", funcName)

	// We'll generate parameter passing separately
	// For now just close the function call
	fmt.Fprintf(w, ")\n")

	// Handle errors first (typically last result)
	errorIndex := -1
	for i, result := range results {
		if result.Type == "error" {
			errorIndex = i
			break
		}
	}

	if errorIndex >= 0 {
		fmt.Fprintf(w, "\tif %s != nil {\n", results[errorIndex].Name)
		fmt.Fprintf(w, "\t\treturn nil, fmt.Errorf(\"%s failed: %%w\", %s)\n", funcName, results[errorIndex].Name)
		fmt.Fprintf(w, "\t}\n")
	}

	// Convert non-error results to Ghoul expressions
	nonErrorResults := make([]ResultConversionInfo, 0, len(results))
	for i, result := range results {
		if i != errorIndex {
			nonErrorResults = append(nonErrorResults, result)
		}
	}

	if len(nonErrorResults) == 0 {
		// No return value, return NIL
		fmt.Fprintf(w, "\treturn e.NIL, nil\n")
	} else if len(nonErrorResults) == 1 {
		// Single return value
		result := nonErrorResults[0]
		fmt.Fprintf(w, "\treturn %s, nil\n", tm.convertValueToExpression(result.Name, result.Type))
	} else {
		// Multiple return values - create a list
		fmt.Fprintf(w, "\treturn ")
		for _, result := range nonErrorResults {
			fmt.Fprintf(w, "e.Cons(%s, ", tm.convertValueToExpression(result.Name, result.Type))
		}
		fmt.Fprintf(w, "e.NIL")
		for range nonErrorResults {
			fmt.Fprint(w, ")")
		}
		fmt.Fprintf(w, ", nil\n")
	}

	return nil
}

// convertValueToExpression converts a Go value to a Ghoul expression
func (tm *TypeMapper) convertValueToExpression(valueName, goType string) string {
	// Check if it's a primitive type
	if ghouType, exists := tm.primitiveMap[goType]; exists {
		switch ghouType {
		case "Boolean":
			return fmt.Sprintf("e.Boolean(%s)", valueName)
		case "Integer":
			return fmt.Sprintf("e.Integer(%s)", valueName)
		case "Float":
			return fmt.Sprintf("e.Float(%s)", valueName)
		case "String":
			return fmt.Sprintf("e.String(%s)", valueName)
		}
	}

	// For complex types, wrap in Foreign
	return fmt.Sprintf("e.Wrapp(%s)", valueName)
}

// GenerateParameterList generates the parameter list for calling the Go function
func (tm *TypeMapper) GenerateParameterList(params []string, w io.Writer) error {
	for i, param := range params {
		if i > 0 {
			fmt.Fprint(w, ", ")
		}
		fmt.Fprint(w, param)
	}
	return nil
}