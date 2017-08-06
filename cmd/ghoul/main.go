package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/archevel/ghoul"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Welcome to Ghoul version 0.1")
		repl()
	} else if len(args) == 1 {
		runFile(args[0])
	}
}

func runFile(path string) {
	f, err := os.Open(path)

	if err == nil {
		g := ghoul.NewGhoul()
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

func repl() {
	g := ghoul.NewGhoul()
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
