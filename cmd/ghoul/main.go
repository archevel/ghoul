//go:generate bash ../../scripts/possess.sh
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/archevel/ghoul"
	"github.com/archevel/ghoul/engraving"
)

func main() {
	var verbose = flag.Bool("v", false, "enable verbose (trace) logging")
	var noPrelude = flag.Bool("noprelude", false, "skip loading the standard prelude")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Welcome to Ghoul version 0.1")
		repl(*verbose, *noPrelude)
	} else if len(args) == 1 {
		runFile(args[0], *verbose, *noPrelude)
	} else {
		fmt.Println("Usage: ghoul [-v] [-noprelude] [file]")
		os.Exit(1)
	}
}

func newGhoul(verbose bool, noPrelude bool) ghoul.Ghoul {
	if noPrelude {
		return ghoul.NewBare()
	}
	if verbose {
		return ghoul.NewLoggingGhoul(engraving.VerboseLogger)
	}
	return ghoul.New()
}

func runFile(path string, verbose bool, noPrelude bool) {
	g := newGhoul(verbose, noPrelude)
	_, processErr := g.ProcessFile(path)
	if processErr != nil {
		fmt.Println(processErr)
		os.Exit(1)
	}
}

func repl(verbose bool, noPrelude bool) {
	g := newGhoul(verbose, noPrelude)
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	for text, readErr := reader.ReadString('\n'); readErr == nil; text, readErr = reader.ReadString('\n') {
		result, err := g.Process(strings.NewReader(text))
		if err != nil {
			fmt.Printf("Error: %s\n\n> ", err)
		} else {
			fmt.Printf("%s\n> ", result.Repr())
		}
	}
}
