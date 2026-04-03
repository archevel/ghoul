package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	prepareCmd := flag.NewFlagSet("prepare", flag.ExitOnError)
	verbose := prepareCmd.Bool("verbose", false, "verbose output during build")
	workDir := prepareCmd.String("work-dir", "", "override build directory (default: .ghoul/build/)")
	keep := prepareCmd.Bool("keep", false, "preserve build directory after build")
	noPrelude := prepareCmd.Bool("no-prelude", false, "omit the standard prelude from the binary")
	noStdlib := prepareCmd.Bool("no-stdlib", false, "don't auto-include default stdlib mummies")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "prepare":
		prepareCmd.Parse(os.Args[2:])
		args := prepareCmd.Args()
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "Usage: undertaker prepare [flags] <binary-name> <graveyard.toml>")
			prepareCmd.PrintDefaults()
			os.Exit(1)
		}

		opts := BuildOptions{
			BinaryName:    args[0],
			GraveyardFile: args[1],
			WorkDir:       *workDir,
			Verbose:       *verbose,
			Keep:          *keep,
			NoPrelude:     *noPrelude,
			NoStdlib:      *noStdlib,
		}

		if err := build(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: undertaker <command> [flags] [arguments]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  prepare <binary-name> <graveyard.toml>   Build a custom ghoul binary")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags for prepare:")
	fmt.Fprintln(os.Stderr, "  --verbose       Verbose output during build")
	fmt.Fprintln(os.Stderr, "  --work-dir      Override build directory (default: .ghoul/build/)")
	fmt.Fprintln(os.Stderr, "  --keep          Preserve build directory after build")
	fmt.Fprintln(os.Stderr, "  --no-prelude    Omit the standard prelude from the binary")
	fmt.Fprintln(os.Stderr, "  --no-stdlib     Don't auto-include default stdlib mummies")
}
