package wraith

import (
	"fmt"
	"go/ast"
	"go/types"
	"golang.org/x/tools/go/packages"
	"strings"
)

// Analyzer analyzes Go packages to extract exportable function information
type Analyzer struct {
	config *Config
}

// FunctionInfo holds information about a discovered function
type FunctionInfo struct {
	Name       string
	Params     []ParameterInfo
	Results    []ParameterInfo
	Doc        string
	Receiver   *ParameterInfo
	IsVariadic bool
	IsGeneric  bool
}

// ParameterInfo holds information about function parameters and return values
type ParameterInfo struct {
	Name string      // Parameter name
	Type types.Type  // Type information
}

type StructInfo struct {
	Name   string
	Fields []FieldInfo
}

type FieldInfo struct {
	Name string
	Type types.Type
}

type InterfaceInfo struct {
	Name    string
	Methods []FunctionInfo
}

type ValueInfo struct {
	Name  string
	Type  types.Type
	IsVar bool
}

type PackageInfo struct {
	Name       string
	ImportPath string
	Functions  []FunctionInfo
	Structs    []StructInfo
	Interfaces []InterfaceInfo
	Values     []ValueInfo
}

// NewAnalyzer creates a new package analyzer
func NewAnalyzer(config *Config) *Analyzer {
	return &Analyzer{config: config}
}

// AnalyzePackage analyzes the configured Go package and returns package information
func (a *Analyzer) AnalyzePackage() (*PackageInfo, error) {
	if a.config.Verbose {
		fmt.Printf("Loading package from: %s\n", a.config.PackagePath)
	}

	// Use go/packages to load the package with full type information
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir: a.config.PackagePath,  // Set working directory to package path
	}

	// Load the package as "." to work relative to the directory
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found at path: %s", a.config.PackagePath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		var errs []string
		for _, err := range pkg.Errors {
			errs = append(errs, err.Error())
		}
		return nil, fmt.Errorf("package has errors: %s", strings.Join(errs, "; "))
	}

	if a.config.Verbose {
		fmt.Printf("Loaded package: %s (%s)\n", pkg.Name, pkg.PkgPath)
		fmt.Printf("Found %d source files\n", len(pkg.Syntax))
	}

	packageInfo := &PackageInfo{
		Name:       pkg.Name,
		ImportPath: pkg.PkgPath,
		Functions:  []FunctionInfo{},
	}

	for _, file := range pkg.Syntax {
		// Skip test files — they can contain constants that duplicate
		// or shadow real package exports
		filename := pkg.Fset.Position(file.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			if a.config.Verbose {
				fmt.Printf("  Skipping test file: %s\n", filename)
			}
			continue
		}
		functions, structs, interfaces, values := a.extractDeclarations(file, pkg)
		packageInfo.Functions = append(packageInfo.Functions, functions...)
		packageInfo.Structs = append(packageInfo.Structs, structs...)
		packageInfo.Interfaces = append(packageInfo.Interfaces, interfaces...)
		packageInfo.Values = append(packageInfo.Values, values...)
	}

	if a.config.Verbose {
		fmt.Printf("Discovered %d exported functions, %d values\n", len(packageInfo.Functions), len(packageInfo.Values))
	}

	return packageInfo, nil
}

func (a *Analyzer) extractDeclarations(file *ast.File, pkg *packages.Package) ([]FunctionInfo, []StructInfo, []InterfaceInfo, []ValueInfo) {
	var functions []FunctionInfo
	var structs []StructInfo
	var interfaces []InterfaceInfo
	var values []ValueInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name.IsExported() {
				function := a.processFunctionDecl(node, pkg)
				if function != nil {
					functions = append(functions, *function)
				}
			}
			return false // don't descend into function bodies
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.IsExported() {
					isGenericType := typeSpec.TypeParams != nil && typeSpec.TypeParams.NumFields() > 0
					if isGenericType {
						continue
					}
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						structInfo := a.processStructType(typeSpec.Name.Name, structType, pkg)
						if structInfo != nil {
							structs = append(structs, *structInfo)
						}
					}
					if ifaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						ifaceInfo := a.processInterfaceType(typeSpec.Name.Name, ifaceType, pkg)
						if ifaceInfo != nil {
							interfaces = append(interfaces, *ifaceInfo)
						}
					}
				}
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					isVar := node.Tok.String() == "var"
					for _, name := range valueSpec.Names {
						if name.IsExported() {
							// Verify the name is in the package scope, not just file-scoped
							obj := pkg.Types.Scope().Lookup(name.Name)
							if obj == nil {
								continue
							}
							valType := pkg.TypesInfo.TypeOf(name)
							values = append(values, ValueInfo{
								Name:  name.Name,
								Type:  valType,
								IsVar: isVar,
							})
						}
					}
				}
			}
		}
		return true
	})

	return functions, structs, interfaces, values
}

func (a *Analyzer) processStructType(name string, structType *ast.StructType, pkg *packages.Package) *StructInfo {
	var fields []FieldInfo
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 || !field.Names[0].IsExported() {
			continue
		}
		fieldType := pkg.TypesInfo.TypeOf(field.Type)
		for _, fieldName := range field.Names {
			if fieldName.IsExported() {
				fields = append(fields, FieldInfo{
					Name: fieldName.Name,
					Type: fieldType,
				})
			}
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return &StructInfo{Name: name, Fields: fields}
}

// processFunctionDecl processes a function declaration and extracts relevant information
func (a *Analyzer) processFunctionDecl(funcDecl *ast.FuncDecl, pkg *packages.Package) *FunctionInfo {
	funcName := funcDecl.Name.Name

	if a.config.Verbose {
		fmt.Printf("  Processing function: %s\n", funcName)
	}

	isGeneric := funcDecl.Type.TypeParams != nil && funcDecl.Type.TypeParams.NumFields() > 0

	doc := ""
	if funcDecl.Doc != nil {
		doc = funcDecl.Doc.Text()
	}

	// Process receiver (for methods)
	var receiver *ParameterInfo
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recv := funcDecl.Recv.List[0]
		recvType := pkg.TypesInfo.TypeOf(recv.Type)
		// Skip methods on unexported types
		if isUnexportedType(recvType) {
			return nil
		}
		receiver = &ParameterInfo{
			Name: getFieldName(recv),
			Type: recvType,
		}
	}

	// Process parameters
	var params []ParameterInfo
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			fieldType := pkg.TypesInfo.TypeOf(field.Type)
			fieldName := getFieldName(field)

			// Handle multiple names for the same type (e.g., "a, b int")
			if len(field.Names) > 1 {
				for _, name := range field.Names {
					params = append(params, ParameterInfo{
						Name: name.Name,
						Type: fieldType,
					})
				}
			} else {
				params = append(params, ParameterInfo{
					Name: fieldName,
					Type: fieldType,
				})
			}
		}
	}

	// Process return values
	var results []ParameterInfo
	if funcDecl.Type.Results != nil {
		for i, field := range funcDecl.Type.Results.List {
			fieldType := pkg.TypesInfo.TypeOf(field.Type)
			fieldName := getFieldName(field)

			// Generate name if not provided
			if fieldName == "" {
				if i == len(funcDecl.Type.Results.List)-1 && isErrorType(fieldType) {
					fieldName = "err"
				} else {
					fieldName = fmt.Sprintf("result%d", i)
				}
			}

			results = append(results, ParameterInfo{
				Name: fieldName,
				Type: fieldType,
			})
		}
	}

	isVariadic := false
	if funcObj := pkg.TypesInfo.ObjectOf(funcDecl.Name); funcObj != nil {
		if sig, ok := funcObj.Type().(*types.Signature); ok {
			isVariadic = sig.Variadic()
		}
	}

	return &FunctionInfo{
		Name:       funcName,
		Params:     params,
		Results:    results,
		Doc:        doc,
		Receiver:   receiver,
		IsVariadic: isVariadic,
		IsGeneric:  isGeneric,
	}
}

func (a *Analyzer) processInterfaceType(name string, ifaceType *ast.InterfaceType, pkg *packages.Package) *InterfaceInfo {
	var methods []FunctionInfo
	for _, method := range ifaceType.Methods.List {
		if len(method.Names) == 0 || !method.Names[0].IsExported() {
			continue
		}
		methodName := method.Names[0].Name
		methodType := pkg.TypesInfo.TypeOf(method.Type)
		sig, ok := methodType.(*types.Signature)
		if !ok {
			continue
		}

		var params []ParameterInfo
		for i := 0; i < sig.Params().Len(); i++ {
			p := sig.Params().At(i)
			params = append(params, ParameterInfo{
				Name: p.Name(),
				Type: p.Type(),
			})
		}

		var results []ParameterInfo
		for i := 0; i < sig.Results().Len(); i++ {
			r := sig.Results().At(i)
			rName := r.Name()
			if rName == "" {
				if isErrorType(r.Type()) {
					rName = "err"
				} else {
					rName = fmt.Sprintf("result%d", i)
				}
			}
			results = append(results, ParameterInfo{
				Name: rName,
				Type: r.Type(),
			})
		}

		methods = append(methods, FunctionInfo{
			Name:    methodName,
			Params:  params,
			Results: results,
		})
	}
	if len(methods) == 0 {
		return nil
	}
	return &InterfaceInfo{Name: name, Methods: methods}
}

// getFieldName extracts the name from a field, handling unnamed fields
func getFieldName(field *ast.Field) string {
	if len(field.Names) > 0 {
		return field.Names[0].Name
	}
	return ""
}

// isErrorType checks if a type is the built-in error interface
func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	return t.String() == "error"
}