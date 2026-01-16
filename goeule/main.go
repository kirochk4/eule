package main

import (
	"bufio"
	"fmt"
	"goeule/eule"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--help" {
		showHelp()
		return
	}

	var err error
	if len(os.Args) < 2 {
		err = runRepl()
	} else {
		err = runFile(os.Args[1:])
	}

	if err != nil {
		log.Fatal(err)
	}
}

func runFile(args []string) error {
	scriptPath := args[0]
	source, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("run file: %w", err)
	}
	return eule.New().Interpret(source)
}

func runRepl() error {
	vm := eule.New()
	fmt.Printf("eule v%s\n", eule.Version)
	fmt.Println("exit using ctrl+c")
	for {
		fmt.Print("> ")
		source, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("run repl: %w", err)
		}
		vm.Interpret(source)
	}
}

func showHelp() {
	fmt.Printf("eule v%s\n", eule.Version)
	fmt.Println()
	fmt.Println("usage:")
	fmt.Println(format("repl", "eule"))
	fmt.Println(format("file", "eule [script] [...arguments]"))
	fmt.Println()
	fmt.Println("optional arguments:")
	fmt.Println(format("--help", "show command line usage"))
	fmt.Println(format("--version", "show version"))
}

func format(arg, desc string) string {
	return fmt.Sprintf("  %-18s%s", arg, desc)
}
