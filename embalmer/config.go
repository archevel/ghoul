package embalmer

import (
	"fmt"
	"os"
	"path/filepath"
)

type MummificationConfig struct {
	PackagePath     string // Filesystem path to the package directory
	ImportPath      string // Go import path (e.g. "math", "github.com/foo/bar") — used instead of PackagePath when set
	WorkDir         string // Directory context for resolving ImportPath (e.g. a temp module dir)
	OutputDir       string
	Verbose         bool
	SkipUnwrappable bool
}

type Config struct {
	PackagePath     string // Filesystem path to the package directory
	ImportPath      string // Go import path — used instead of PackagePath when set
	WorkDir         string // Directory context for resolving ImportPath
	OutputFile      string
	PackageName     string
	Verbose         bool
	SkipUnwrappable bool
}

func Mummify(config *MummificationConfig) error {
	var packageName string

	if config.ImportPath != "" {
		// Import-path mode: derive package name from the import path
		packageName = filepath.Base(config.ImportPath)
	} else {
		if _, err := os.Stat(config.PackagePath); os.IsNotExist(err) {
			return fmt.Errorf("package path does not exist: %s", config.PackagePath)
		}
		packageName = filepath.Base(config.PackagePath)
	}

	mummyName := packageName + "_mummy"

	outputDir := config.OutputDir
	if outputDir == "" {
		if config.ImportPath != "" {
			return fmt.Errorf("OutputDir is required when using ImportPath")
		}
		outputDir = filepath.Join(filepath.Dir(config.PackagePath), mummyName)
	}

	outputFile := filepath.Join(outputDir, packageName+".go")

	if config.Verbose {
		fmt.Printf("📦 Target package: %s\n", packageName)
		fmt.Printf("⚰️  Generating mummy: %s\n", outputDir)
	}

	legacyConfig := &Config{
		PackagePath:     config.PackagePath,
		ImportPath:      config.ImportPath,
		WorkDir:         config.WorkDir,
		OutputFile:      outputFile,
		PackageName:     mummyName,
		Verbose:         config.Verbose,
		SkipUnwrappable: config.SkipUnwrappable,
	}

	return GenerateWrappers(legacyConfig)
}

func GenerateWrappers(config *Config) error {
	analyzer := NewAnalyzer(config)

	packageInfo, err := analyzer.AnalyzePackage()
	if err != nil {
		return err
	}

	generator, err := NewGenerator(config)
	if err != nil {
		return err
	}
	return generator.GenerateCode(packageInfo)
}
