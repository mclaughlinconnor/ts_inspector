package main

import (
	"fmt"

	"github.com/mclaughlinconnor/ts_inspector/parsers/angular_content"
	"github.com/mclaughlinconnor/ts_inspector/parsers/javascript"
	"github.com/mclaughlinconnor/ts_inspector/parsers/pug"
	"github.com/mclaughlinconnor/ts_inspector/parsers/typescript"
	sitter "github.com/smacker/go-tree-sitter"
)

func main() {
	input := []byte("span Hello World\n")

	parser := sitter.NewParser()
	parser.SetLanguage(pug.GetLanguage())
	parser.SetLanguage(angular_content.GetLanguage())
	parser.SetLanguage(typescript.GetLanguage())
	parser.SetLanguage(javascript.GetLanguage())

	tree := parser.Parse(nil, input)

	n := tree.RootNode()

	fmt.Println("AST:", n)
	fmt.Println("Root type:", n.Type())
	fmt.Println("Root children:", n.ChildCount())
}
