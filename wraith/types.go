package wraith

import (
	"fmt"
	"go/types"
	"io"
	"strings"
	"text/template"
)

type TypeMapper struct {
	primitiveMap map[string]string
	templates    map[string]*template.Template
}

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
			"float32":       "Float",
			"float64":       "Float",
			"untyped int":   "Integer",
			"untyped float": "Float",
			"untyped string": "String",
			"untyped bool":  "Boolean",
		},
		templates: make(map[string]*template.Template),
	}

	err := tm.initializeTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize type mapping templates: %w", err)
	}

	return tm, nil
}

type FuncParamInfo struct {
	Type      string
	GhoulType string // e.g. "Integer", "String", empty for complex
}

type FuncSignatureInfo struct {
	Params  []FuncParamInfo
	Results []FuncParamInfo
}

type ArgConversionInfo struct {
	N             int
	Name          string
	Type          string
	BuiltInType   string
	FuncSignature *FuncSignatureInfo
	IsVariadic    bool
}

type ResultConversionInfo struct {
	Index int
	Type  string
	Name  string
}

func (tm *TypeMapper) initializeTemplates() error {
	// Built-in types use Node kind checks and direct field access
	builtInTypeTemplate := `
	ghoulArg_{{.Name}} := args[argIdx]
	if ghoulArg_{{.Name}}.Kind != _e.{{.BuiltInType | kindConst}} {
		return nil, _fmt.Errorf("expected {{.BuiltInType | lower}} for parameter '{{.Name}}', got %s", _e.NodeTypeName(ghoulArg_{{.Name}}))
	}
	param_{{.Name}} := {{.Type}}(ghoulArg_{{.Name}}.{{.BuiltInType | fieldName}})
	argIdx++
`

	// Foreign (mummy) types extract the Go value from the MummyNode's ForeignVal
	foreignTypeTemplate := `
	ghoulArg_{{.Name}} := args[argIdx]
	if ghoulArg_{{.Name}}.Kind != _e.MummyNode {
		return nil, _fmt.Errorf("expected mummy for parameter '{{.Name}}', got %s", _e.NodeTypeName(ghoulArg_{{.Name}}))
	}
	var param_{{.Name}} {{.Type}}
	if ghoulArg_{{.Name}}.ForeignVal != nil {
		var ok bool
		param_{{.Name}}, ok = ghoulArg_{{.Name}}.ForeignVal.({{.Type}})
		if !ok {
			param_{{.Name}}_ptr, ok := ghoulArg_{{.Name}}.ForeignVal.(*{{.Type}})
			if !ok {
				return nil, _fmt.Errorf("parameter '{{.Name}}': mummy contains %T, expected {{.Type}}", ghoulArg_{{.Name}}.ForeignVal)
			}
			param_{{.Name}} = *param_{{.Name}}_ptr
		}
	}
	argIdx++
`

	funcMap := template.FuncMap{
		"lower": func(s string) string {
			switch s {
			case "Integer":
				return "integer"
			case "String":
				return "string"
			case "Boolean":
				return "boolean"
			case "Float":
				return "float"
			default:
				return s
			}
		},
		"kindConst": func(s string) string {
			switch s {
			case "Integer":
				return "IntegerNode"
			case "String":
				return "StringNode"
			case "Boolean":
				return "BooleanNode"
			case "Float":
				return "FloatNodeKind"
			default:
				return s
			}
		},
		"fieldName": func(s string) string {
			switch s {
			case "Integer":
				return "IntVal"
			case "String":
				return "StrVal"
			case "Boolean":
				return "BoolVal"
			case "Float":
				return "FloatVal"
			default:
				return s
			}
		},
	}

	var err error
	tm.templates["builtin"], err = template.New("builtin").Funcs(funcMap).Parse(builtInTypeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse builtin type template: %w", err)
	}

	tm.templates["foreign"], err = template.New("foreign").Parse(foreignTypeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse foreign type template: %w", err)
	}

	return nil
}

func (tm *TypeMapper) GenerateArgumentConversion(info ArgConversionInfo, w io.Writer) error {
	if info.IsVariadic {
		return tm.generateVariadicConversion(info, w)
	}
	if info.FuncSignature != nil {
		return tm.generateFunctionAdapter(info, w)
	}
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

func (tm *TypeMapper) generateVariadicConversion(info ArgConversionInfo, w io.Writer) error {
	name := info.Name
	varName := "param_" + name

	fmt.Fprintf(w, "\tvar %s %s\n", varName, info.Type)
	fmt.Fprintf(w, "\tfor argIdx < len(args) {\n")

	if info.BuiltInType != "" {
		elemType := strings.TrimPrefix(info.Type, "[]")
		kindConst := builtInKindConst(info.BuiltInType)
		fieldName := builtInFieldName(info.BuiltInType)
		fmt.Fprintf(w, "\t\tghoulElem := args[argIdx]\n")
		fmt.Fprintf(w, "\t\tif ghoulElem.Kind != _e.%s {\n", kindConst)
		fmt.Fprintf(w, "\t\t\treturn nil, _fmt.Errorf(\"%s: expected %s, got %%s\", _e.NodeTypeName(ghoulElem))\n",
			name, strings.ToLower(info.BuiltInType))
		fmt.Fprintf(w, "\t\t}\n")
		fmt.Fprintf(w, "\t\t%s = append(%s, %s(ghoulElem.%s))\n", varName, varName, elemType, fieldName)
	} else {
		elemType := strings.TrimPrefix(info.Type, "[]")
		fmt.Fprintf(w, "\t\tghoulElem := args[argIdx]\n")
		fmt.Fprintf(w, "\t\tif ghoulElem.Kind != _e.MummyNode {\n")
		fmt.Fprintf(w, "\t\t\treturn nil, _fmt.Errorf(\"%s: expected mummy, got %%s\", _e.NodeTypeName(ghoulElem))\n", name)
		fmt.Fprintf(w, "\t\t}\n")
		fmt.Fprintf(w, "\t\telem, ok := ghoulElem.ForeignVal.(%s)\n", elemType)
		fmt.Fprintf(w, "\t\tif !ok {\n")
		fmt.Fprintf(w, "\t\t\treturn nil, _fmt.Errorf(\"%s: mummy contains %%T, expected %s\", ghoulElem.ForeignVal)\n", name, elemType)
		fmt.Fprintf(w, "\t\t}\n")
		fmt.Fprintf(w, "\t\t%s = append(%s, elem)\n", varName, varName)
	}

	fmt.Fprintf(w, "\t\targIdx++\n")
	fmt.Fprintf(w, "\t}\n")
	return nil
}

func (tm *TypeMapper) generateFunctionAdapter(info ArgConversionInfo, w io.Writer) error {
	sig := info.FuncSignature
	name := info.Name

	// Assert the argument is a Ghoul Function (stored as a FunctionNode)
	fmt.Fprintf(w, "\tghoulArg_%s := args[argIdx]\n", name)
	fmt.Fprintf(w, "\tif ghoulArg_%s.FuncVal == nil {\n", name)
	fmt.Fprintf(w, "\t\treturn nil, _fmt.Errorf(\"expected function for parameter '%s', got %%s\", _e.NodeTypeName(ghoulArg_%s))\n", name, name)
	fmt.Fprintf(w, "\t}\n")
	fmt.Fprintf(w, "\tghoulFunc_%s := ghoulArg_%s.FuncVal\n", name, name)

	// Build the Go function adapter
	fmt.Fprintf(w, "\tparam_%s := %s{\n", name, info.Type)

	// Build ghoul argument slice from Go parameters
	fmt.Fprintf(w, "\t\tghoulArgs := make([]*_e.Node, %d)\n", len(sig.Params))
	for i, p := range sig.Params {
		if p.GhoulType != "" {
			fmt.Fprintf(w, "\t\tghoulArgs[%d] = _e.%s(%s(p%d))\n", i, nodeConstructor(p.GhoulType), nodeConstructorCast(p.GhoulType), i)
		} else {
			fmt.Fprintf(w, "\t\tghoulArgs[%d] = _e.MummyNodeVal(p%d, \"%s\")\n", i, i, p.Type)
		}
	}

	if len(sig.Results) == 0 {
		fmt.Fprintf(w, "\t\t(*ghoulFunc_%s)(ghoulArgs, ev)\n", name)
	} else if len(sig.Results) == 1 {
		fmt.Fprintf(w, "\t\tresult, _ := (*ghoulFunc_%s)(ghoulArgs, ev)\n", name)
		r := sig.Results[0]
		fmt.Fprintf(w, "\t\treturn %s\n", tm.ghoulToGoConversion("result", r))
	} else {
		fmt.Fprintf(w, "\t\tresult, _ := (*ghoulFunc_%s)(ghoulArgs, ev)\n", name)
		for i, r := range sig.Results {
			varName := fmt.Sprintf("goResult%d", i)
			fmt.Fprintf(w, "\t\t%s := %s\n", varName, tm.ghoulToGoConversion(fmt.Sprintf("result.Children[%d]", i), r))
		}
		fmt.Fprintf(w, "\t\treturn ")
		for i := range sig.Results {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, "goResult%d", i)
		}
		fmt.Fprint(w, "\n")
	}

	fmt.Fprintf(w, "\t}\n")
	fmt.Fprintf(w, "\targIdx++\n")
	return nil
}

func (tm *TypeMapper) ghoulToGoConversion(exprVar string, param FuncParamInfo) string {
	switch param.GhoulType {
	case "Integer":
		return fmt.Sprintf("%s(%s.IntVal)", param.Type, exprVar)
	case "String":
		return fmt.Sprintf("%s(%s.StrVal)", param.Type, exprVar)
	case "Boolean":
		return fmt.Sprintf("%s(%s.BoolVal)", param.Type, exprVar)
	case "Float":
		return fmt.Sprintf("%s(%s.FloatVal)", param.Type, exprVar)
	default:
		return fmt.Sprintf("%s.ForeignVal.(%s)", exprVar, param.Type)
	}
}

func (tm *TypeMapper) MapGoTypeToGhoul(goType types.Type) (ghoulType string, isForeign bool) {
	typeStr := goType.String()

	if ghoulType, exists := tm.primitiveMap[typeStr]; exists {
		return ghoulType, false
	}

	// Handle type aliases like `type Score int` by checking the underlying type
	if ghoulType, exists := tm.primitiveMap[goType.Underlying().String()]; exists {
		return ghoulType, false
	}

	if ptr, ok := goType.(*types.Pointer); ok {
		if ghoulType, exists := tm.primitiveMap[ptr.Elem().String()]; exists {
			return ghoulType, false
		}
		if ghoulType, exists := tm.primitiveMap[ptr.Elem().Underlying().String()]; exists {
			return ghoulType, false
		}
	}

	if slice, ok := goType.(*types.Slice); ok {
		if tm.isPrimitiveType(slice.Elem()) {
			return "", true
		}
	}

	return "", true
}

func (tm *TypeMapper) IsFunction(goType types.Type) bool {
	_, ok := goType.Underlying().(*types.Signature)
	return ok
}

func (tm *TypeMapper) BuildFuncSignature(goType types.Type) *FuncSignatureInfo {
	sig, ok := goType.Underlying().(*types.Signature)
	if !ok {
		return nil
	}

	info := &FuncSignatureInfo{}

	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		p := params.At(i)
		ghoulType, _ := tm.MapGoTypeToGhoul(p.Type())
		info.Params = append(info.Params, FuncParamInfo{
			Type:      qualifiedTypeToAlias(p.Type().String()),
			GhoulType: ghoulType,
		})
	}

	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		r := results.At(i)
		ghoulType, _ := tm.MapGoTypeToGhoul(r.Type())
		info.Results = append(info.Results, FuncParamInfo{
			Type:      qualifiedTypeToAlias(r.Type().String()),
			GhoulType: ghoulType,
		})
	}

	return info
}

func qualifiedTypeToAlias(typeStr string) string {
	prefix := ""
	inner := typeStr
	for strings.HasPrefix(inner, "*") {
		prefix += "*"
		inner = inner[1:]
	}
	if strings.HasPrefix(inner, "[]") {
		prefix += "[]"
		inner = inner[2:]
		// Strip pointers after slice prefix (e.g. "[]*net/http.Cookie")
		for strings.HasPrefix(inner, "*") {
			prefix += "*"
			inner = inner[1:]
		}
	}

	// Handle map types: map[K]V where K and V may be qualified
	if strings.HasPrefix(inner, "map[") {
		bracketEnd := findMatchingBracket(inner, 3)
		if bracketEnd > 0 {
			keyType := qualifiedTypeToAlias(inner[4:bracketEnd])
			valType := qualifiedTypeToAlias(inner[bracketEnd+1:])
			return prefix + "map[" + keyType + "]" + valType
		}
	}

	// Handle channel types: chan T, chan<- T, <-chan T
	if strings.HasPrefix(inner, "chan<- ") {
		return prefix + "chan<- " + qualifiedTypeToAlias(strings.TrimSpace(inner[6:]))
	}
	if strings.HasPrefix(inner, "<-chan ") {
		return prefix + "<-chan " + qualifiedTypeToAlias(strings.TrimSpace(inner[6:]))
	}
	if strings.HasPrefix(inner, "chan ") {
		return prefix + "chan " + qualifiedTypeToAlias(strings.TrimSpace(inner[4:]))
	}

	lastSlash := strings.LastIndex(inner, "/")
	if lastSlash >= 0 {
		inner = inner[lastSlash+1:]
	}
	return prefix + inner
}

func findMatchingBracket(s string, openPos int) int {
	depth := 1
	for i := openPos + 1; i < len(s); i++ {
		if s[i] == '[' {
			depth++
		} else if s[i] == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// UnsupportedTypeReason returns a description of why a type can't be wrapped,
// or empty string if the type is supported.
func (tm *TypeMapper) UnsupportedTypeReason(t types.Type) string {
	if t == nil {
		return ""
	}
	switch underlying := t.Underlying().(type) {
	case *types.Signature:
		if underlying.Variadic() {
			return fmt.Sprintf("variadic function type %s", t)
		}
	}
	return ""
}

func (tm *TypeMapper) isPrimitiveType(t types.Type) bool {
	_, exists := tm.primitiveMap[t.String()]
	return exists
}

func (tm *TypeMapper) convertValueToExpression(valueName, goType string) string {
	if ghoulType, exists := tm.primitiveMap[goType]; exists {
		switch ghoulType {
		case "Boolean":
			return fmt.Sprintf("_e.BoolNode(%s)", valueName)
		case "Integer":
			return fmt.Sprintf("_e.IntNode(int64(%s))", valueName)
		case "Float":
			return fmt.Sprintf("_e.FloatNode(float64(%s))", valueName)
		case "String":
			return fmt.Sprintf("_e.StrNode(string(%s))", valueName)
		}
	}

	return fmt.Sprintf("_e.MummyNodeVal(%s, \"%s\")", valueName, goType)
}

// builtInKindConst returns the Node kind constant name for a built-in ghoul type
func builtInKindConst(ghoulType string) string {
	switch ghoulType {
	case "Integer":
		return "IntegerNode"
	case "String":
		return "StringNode"
	case "Boolean":
		return "BooleanNode"
	case "Float":
		return "FloatNodeKind"
	default:
		return ghoulType
	}
}

// builtInFieldName returns the Node field name for a built-in ghoul type
func builtInFieldName(ghoulType string) string {
	switch ghoulType {
	case "Integer":
		return "IntVal"
	case "String":
		return "StrVal"
	case "Boolean":
		return "BoolVal"
	case "Float":
		return "FloatVal"
	default:
		return ghoulType
	}
}

// nodeConstructorCast returns the Go cast needed for a node constructor
func nodeConstructorCast(ghoulType string) string {
	switch ghoulType {
	case "Integer":
		return "int64"
	case "Float":
		return "float64"
	case "String":
		return "string"
	case "Boolean":
		return "bool"
	default:
		return ""
	}
}

// nodeConstructor returns the e.XNode constructor name for a ghoul type
func nodeConstructor(ghoulType string) string {
	switch ghoulType {
	case "Integer":
		return "IntNode"
	case "String":
		return "StrNode"
	case "Boolean":
		return "BoolNode"
	case "Float":
		return "FloatNode"
	default:
		return ghoulType
	}
}
