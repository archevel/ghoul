package wraith

import (
	"fmt"
	"os"
	"path/filepath"
)

type PossessionConfig struct {
	PackagePath     string
	OutputDir       string
	Verbose         bool
	SkipUnwrappable bool
}

type Config struct {
	PackagePath     string
	OutputFile      string
	PackageName     string
	Verbose         bool
	SkipUnwrappable bool
}

func PossessPackage(config *PossessionConfig) error {
	if _, err := os.Stat(config.PackagePath); os.IsNotExist(err) {
		return fmt.Errorf("package path does not exist: %s", config.PackagePath)
	}

	packageName := filepath.Base(config.PackagePath)
	mummyName := packageName + "_mummy"

	outputDir := config.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(filepath.Dir(config.PackagePath), mummyName)
	}

	outputFile := filepath.Join(outputDir, packageName+".go")

	if config.Verbose {
		fmt.Printf("📦 Target package: %s\n", packageName)
		fmt.Printf("⚰️  Generating mummy: %s\n", outputDir)
	}

	legacyConfig := &Config{
		PackagePath:     config.PackagePath,
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
