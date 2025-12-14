package wraith

import (
	"fmt"
	"os"
	"path/filepath"
)

// PossessionConfig holds the configuration for wraith package possession
type PossessionConfig struct {
	// PackagePath is the path to the Go package to possess
	PackagePath string

	// Verbose enables verbose output during possession
	Verbose bool
}

// Config holds the configuration for wraith code generation (legacy)
type Config struct {
	// PackagePath is the path to the Go package to analyze
	PackagePath string

	// OutputFile is the path where generated wrappers will be written
	OutputFile string

	// PackageName is the name of the generated Go package
	PackageName string

	// Verbose enables verbose output during generation
	Verbose bool
}

// PossessPackage is the main entry point for possessing a Go package
// and wrapping it like a mummy for Ghoul use
func PossessPackage(config *PossessionConfig) error {
	// Validate package path exists
	if _, err := os.Stat(config.PackagePath); os.IsNotExist(err) {
		return fmt.Errorf("package path does not exist: %s", config.PackagePath)
	}

	// Get package name from the directory
	packageName := filepath.Base(config.PackagePath)

	// Generate mummy filename
	mummyFilename := packageName + "_mummy.go"
	mummyPath := filepath.Join(config.PackagePath, mummyFilename)

	// Check if mummy file already exists (possession should fail)
	if _, err := os.Stat(mummyPath); !os.IsNotExist(err) {
		return fmt.Errorf("package is already possessed - %s exists (possession failed)", mummyFilename)
	}

	if config.Verbose {
		fmt.Printf("📦 Target package: %s\n", packageName)
		fmt.Printf("🗞️ Generating mummy: %s\n", mummyFilename)
	}

	// Create legacy config for the generator
	legacyConfig := &Config{
		PackagePath: config.PackagePath,
		OutputFile:  mummyPath,
		PackageName: packageName + "_mummy",
		Verbose:     config.Verbose,
	}

	// Generate the mummy wrapper
	return GenerateWrappers(legacyConfig)
}

// GenerateWrappers is the main entry point for generating Ghoul wrappers
// from a Go package (legacy interface)
func GenerateWrappers(config *Config) error {
	analyzer := NewAnalyzer(config)

	// Analyze the package to discover exported functions
	packageInfo, err := analyzer.AnalyzePackage()
	if err != nil {
		return err
	}

	// Generate wrapper code
	generator, err := NewGenerator(config)
	if err != nil {
		return err
	}
	return generator.GenerateCode(packageInfo)
}