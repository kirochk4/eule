package main

import (
	"fmt"
	"goeule/eule"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: eule [script]")
	} else {
		RunFile(os.Args[1])
	}
}

func RunFile(file string) {
	source, _ := os.ReadFile(file)
	vm := eule.New()
	vm.Interpret(source)
}
