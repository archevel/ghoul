package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/archevel/ghoul/embalmer"
)

func main() {
	var (
		verbose         = flag.Bool("v", false, "Enable verbose output")
		skipUnwrappable = flag.Bool("skip-unwrappable", false, "Skip functions that can't be wrapped instead of failing")
		outputDir       = flag.String("o", "", "Output directory for generated mummy (default: next to package)")
	)
	flag.Parse()

	args := flag.Args()

	// Parse command
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: embalmer mummify <package-path> [-v]\n")
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  mummify    Mummify a Go package for Ghoul use\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  -v         Enable verbose output\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  embalmer mummify ./mypackage\n")
		os.Exit(1)
	}

	command := args[0]
	packagePath := args[1]

	if command != "mummify" {
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'. Use 'mummify' to wrap a package.\n", command)
		os.Exit(1)
	}

	// Validate package path
	if packagePath == "" {
		fmt.Fprintf(os.Stderr, "Error: package path is required\n")
		os.Exit(1)
	}

	// Make package path absolute
	absPackagePath, err := filepath.Abs(packagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid package path: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("🧟 Embalmer is mummifying package: %s\n", absPackagePath)
	}

	err = embalmer.Mummify(&embalmer.MummificationConfig{
		PackagePath:     absPackagePath,
		OutputDir:       *outputDir,
		Verbose:         *verbose,
		SkipUnwrappable: *skipUnwrappable,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "💀 Mummification failed: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("🎭 Package successfully mummified!\n")
	}
}
