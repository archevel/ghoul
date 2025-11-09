package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/archevel/ghoul"
	"github.com/archevel/ghoul/logging"
)

func main() {
	var verbose = flag.Bool("v", false, "enable verbose (trace) logging")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Welcome to Ghoul version 0.1")
		repl(*verbose)
	} else if len(args) == 1 {
		runFile(args[0], *verbose)
	} else {
		fmt.Println("Usage: ghoul [-v] [file]")
		os.Exit(1)
	}
}

func runFile(path string, verbose bool) {
	f, err := os.Open(path)

	if err == nil {
		var g ghoul.Ghoul
		if verbose {
			g = ghoul.NewLoggingGhoul(logging.VerboseLogger)
		} else {
			g = ghoul.New()
		}
		_, processErr := g.Process(f)
		f.Close()
		if processErr != nil {
			fmt.Println(processErr)
			os.Exit(1)
		}
	} else {
		fmt.Println(err)
		os.Exit(1)
	}

}

func repl(verbose bool) {
	var g ghoul.Ghoul
	if verbose {
		g = ghoul.NewLoggingGhoul(logging.VerboseLogger)
	} else {
		g = ghoul.New()
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	for text, readErr := reader.ReadString('\n'); readErr == nil; text, readErr = reader.ReadString('\n') {
		res, err := g.Process(strings.NewReader(text))
		if err != nil {
			fmt.Printf("Error: %s\n\n> ", err)
		} else {
			fmt.Printf("%s\n> ", res.Repr())
		}
	}

}
