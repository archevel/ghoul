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
	builtInTypeTemplate := `
	ghoulArg_{{.Name}} := args.First()
	{{.Name}}_val, ok := ghoulArg_{{.Name}}.(e.{{.BuiltInType}})
	if !ok {
		return nil, fmt.Errorf("expected {{.BuiltInType | lower}} for parameter '{{.Name}}', got %s", e.TypeName(ghoulArg_{{.Name}}))
	}
	{{.Name}} := {{.Type}}({{.Name}}_val)
	args, _ = args.Tail()
`

	foreignTypeTemplate := `
	ghoulArg_{{.Name}} := args.First()
	mummy_{{.Name}}, ok := ghoulArg_{{.Name}}.(*mummy.Mummy)
	if !ok {
		return nil, fmt.Errorf("expected mummy for parameter '{{.Name}}', got %s", e.TypeName(ghoulArg_{{.Name}}))
	}
	var {{.Name}} {{.Type}}
	if mummy_{{.Name}}.Unwrap() != nil {
		{{.Name}}, ok = mummy_{{.Name}}.Unwrap().({{.Type}})
		if !ok {
			{{.Name}}_ptr, ok := mummy_{{.Name}}.Unwrap().(*{{.Type}})
			if !ok {
				return nil, fmt.Errorf("parameter '{{.Name}}': mummy contains %T, expected {{.Type}}", mummy_{{.Name}}.Unwrap())
			}
			{{.Name}} = *{{.Name}}_ptr
		}
	}
	args, _ = args.Tail()
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

	fmt.Fprintf(w, "\tvar %s %s\n", name, info.Type)
	fmt.Fprintf(w, "\tfor args != e.NIL {\n")

	if info.BuiltInType != "" {
		elemType := strings.TrimPrefix(info.Type, "[]")
		fmt.Fprintf(w, "\t\tghoulElem := args.First()\n")
		fmt.Fprintf(w, "\t\telemVal, ok := ghoulElem.(e.%s)\n", info.BuiltInType)
		fmt.Fprintf(w, "\t\tif !ok {\n")
		fmt.Fprintf(w, "\t\t\treturn nil, fmt.Errorf(\"%s: expected %s, got %%s\", e.TypeName(ghoulElem))\n",
			name, strings.ToLower(info.BuiltInType))
		fmt.Fprintf(w, "\t\t}\n")
		fmt.Fprintf(w, "\t\t%s = append(%s, %s(elemVal))\n", name, name, elemType)
	} else {
		elemType := strings.TrimPrefix(info.Type, "[]")
		fmt.Fprintf(w, "\t\tghoulElem := args.First()\n")
		fmt.Fprintf(w, "\t\tmummyElem, ok := ghoulElem.(*mummy.Mummy)\n")
		fmt.Fprintf(w, "\t\tif !ok {\n")
		fmt.Fprintf(w, "\t\t\treturn nil, fmt.Errorf(\"%s: expected mummy, got %%s\", e.TypeName(ghoulElem))\n", name)
		fmt.Fprintf(w, "\t\t}\n")
		fmt.Fprintf(w, "\t\telem, ok := mummyElem.Unwrap().(%s)\n", elemType)
		fmt.Fprintf(w, "\t\tif !ok {\n")
		fmt.Fprintf(w, "\t\t\treturn nil, fmt.Errorf(\"%s: mummy contains %%T, expected %s\", mummyElem.Unwrap())\n", name, elemType)
		fmt.Fprintf(w, "\t\t}\n")
		fmt.Fprintf(w, "\t\t%s = append(%s, elem)\n", name, name)
	}

	fmt.Fprintf(w, "\t\targs, _ = args.Tail()\n")
	fmt.Fprintf(w, "\t}\n")
	return nil
}

func (tm *TypeMapper) generateFunctionAdapter(info ArgConversionInfo, w io.Writer) error {
	sig := info.FuncSignature
	name := info.Name

	// Assert the argument is a Ghoul Function
	fmt.Fprintf(w, "\tghoulArg_%s := args.First()\n", name)
	fmt.Fprintf(w, "\tghoulFunc_%s, ok := ghoulArg_%s.(ghoulEval.Function)\n", name, name)
	fmt.Fprintf(w, "\tif !ok {\n")
	fmt.Fprintf(w, "\t\treturn nil, fmt.Errorf(\"expected function for parameter '%s', got %%s\", e.TypeName(ghoulArg_%s))\n", name, name)
	fmt.Fprintf(w, "\t}\n")

	// Build the Go function adapter signature
	fmt.Fprintf(w, "\t%s := %s{\n", name, info.Type)

	// Build ghoul argument list from Go parameters (in reverse to build cons list)
	fmt.Fprintf(w, "\t\tvar ghoulArgs e.List = e.NIL\n")
	for i := len(sig.Params) - 1; i >= 0; i-- {
		p := sig.Params[i]
		if p.GhoulType != "" {
			fmt.Fprintf(w, "\t\tghoulArgs = e.Cons(e.%s(p%d), ghoulArgs)\n", p.GhoulType, i)
		} else {
			fmt.Fprintf(w, "\t\tghoulArgs = e.Cons(mummy.Entomb(p%d, \"%s\"), ghoulArgs)\n", i, p.Type)
		}
	}

	if len(sig.Results) == 0 {
		fmt.Fprintf(w, "\t\t(*ghoulFunc_%s.Fun)(ghoulArgs, ev)\n", name)
	} else if len(sig.Results) == 1 {
		fmt.Fprintf(w, "\t\tresult, _ := (*ghoulFunc_%s.Fun)(ghoulArgs, ev)\n", name)
		r := sig.Results[0]
		fmt.Fprintf(w, "\t\treturn %s\n", tm.ghoulToGoConversion("result", r))
	} else {
		fmt.Fprintf(w, "\t\tresult, _ := (*ghoulFunc_%s.Fun)(ghoulArgs, ev)\n", name)
		for i, r := range sig.Results {
			if i == 0 {
				fmt.Fprintf(w, "\t\tresultList := result.(e.List)\n")
			}
			varName := fmt.Sprintf("goResult%d", i)
			fmt.Fprintf(w, "\t\t%s := %s\n", varName, tm.ghoulToGoConversion("resultList.First()", r))
			if i < len(sig.Results)-1 {
				fmt.Fprintf(w, "\t\tresultList, _ = resultList.Tail()\n")
			}
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
	fmt.Fprintf(w, "\targs, _ = args.Tail()\n")
	return nil
}

func (tm *TypeMapper) ghoulToGoConversion(exprVar string, param FuncParamInfo) string {
	switch param.GhoulType {
	case "Integer":
		return fmt.Sprintf("%s(%s.(e.Integer))", param.Type, exprVar)
	case "String":
		return fmt.Sprintf("%s(%s.(e.String))", param.Type, exprVar)
	case "Boolean":
		return fmt.Sprintf("%s(%s.(e.Boolean))", param.Type, exprVar)
	case "Float":
		return fmt.Sprintf("%s(%s.(e.Float))", param.Type, exprVar)
	default:
		return fmt.Sprintf("%s.(*mummy.Mummy).Unwrap().(%s)", exprVar, param.Type)
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
			return fmt.Sprintf("e.Boolean(%s)", valueName)
		case "Integer":
			return fmt.Sprintf("e.Integer(%s)", valueName)
		case "Float":
			return fmt.Sprintf("e.Float(%s)", valueName)
		case "String":
			return fmt.Sprintf("e.String(%s)", valueName)
		}
	}

	return fmt.Sprintf("mummy.Entomb(%s, \"%s\")", valueName, goType)
}

