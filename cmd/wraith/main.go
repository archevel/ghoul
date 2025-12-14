package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/archevel/ghoul/wraith"
)

func main() {
	var (
		verbose = flag.Bool("v", false, "Enable verbose output")
	)
	flag.Parse()

	args := flag.Args()

	// Parse command
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: wraith possess <package-path> [-v]\n")
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  possess    Wrap a Go package like a mummy for Ghoul use\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  -v         Enable verbose output\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  wraith possess ./mypackage\n")
		os.Exit(1)
	}

	command := args[0]
	packagePath := args[1]

	if command != "possess" {
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'. Use 'possess' to wrap a package.\n", command)
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
		fmt.Printf("🧟 Wraith is possessing package: %s\n", absPackagePath)
	}

	err = wraith.PossessPackage(&wraith.PossessionConfig{
		PackagePath: absPackagePath,
		Verbose:     *verbose,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "💀 Possession failed: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("🎭 Package successfully possessed and wrapped like a mummy!\n")
	}
}