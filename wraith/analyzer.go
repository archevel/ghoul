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
	Name       string           // Function name
	Params     []ParameterInfo  // Parameter information
	Results    []ParameterInfo  // Return value information
	Doc        string           // Documentation comment
	Receiver   *ParameterInfo   // Receiver for methods (nil for functions)
}

// ParameterInfo holds information about function parameters and return values
type ParameterInfo struct {
	Name string      // Parameter name
	Type types.Type  // Type information
}

// PackageInfo holds all discovered information about a package
type PackageInfo struct {
	Name      string          // Package name
	ImportPath string         // Import path
	Functions []FunctionInfo  // All exported functions
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

	// Analyze each source file in the package
	for _, file := range pkg.Syntax {
		functions := a.extractFunctions(file, pkg)
		packageInfo.Functions = append(packageInfo.Functions, functions...)
	}

	if a.config.Verbose {
		fmt.Printf("Discovered %d exported functions\n", len(packageInfo.Functions))
	}

	return packageInfo, nil
}

// extractFunctions extracts all exported functions from an AST file
func (a *Analyzer) extractFunctions(file *ast.File, pkg *packages.Package) []FunctionInfo {
	var functions []FunctionInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name.IsExported() {
				function := a.processFunctionDecl(node, pkg)
				if function != nil {
					functions = append(functions, *function)
				}
			}
		}
		return true
	})

	return functions
}

// processFunctionDecl processes a function declaration and extracts relevant information
func (a *Analyzer) processFunctionDecl(funcDecl *ast.FuncDecl, pkg *packages.Package) *FunctionInfo {
	funcName := funcDecl.Name.Name

	if a.config.Verbose {
		fmt.Printf("  Processing function: %s\n", funcName)
	}

	// Get function documentation
	doc := ""
	if funcDecl.Doc != nil {
		doc = funcDecl.Doc.Text()
	}

	// Process receiver (for methods)
	var receiver *ParameterInfo
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recv := funcDecl.Recv.List[0]
		recvType := pkg.TypesInfo.TypeOf(recv.Type)
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

	return &FunctionInfo{
		Name:     funcName,
		Params:   params,
		Results:  results,
		Doc:      doc,
		Receiver: receiver,
	}
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