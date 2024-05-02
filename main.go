package main

import (
	"fmt"

	"github.com/mclaughlinconnor/ts_inspector/parsers/pug"
	sitter "github.com/smacker/go-tree-sitter"
)

func main() {
	input := []byte("span Hello World\n")

	parser := sitter.NewParser()
	parser.SetLanguage(pug.GetLanguage())

	tree := parser.Parse(nil, input)

	n := tree.RootNode()

	fmt.Println("AST:", n)
	fmt.Println("Root type:", n.Type())
	fmt.Println("Root children:", n.ChildCount())
}
