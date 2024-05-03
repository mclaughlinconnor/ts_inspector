package main

import (
	"context"
	"log"

	"github.com/mclaughlinconnor/ts_inspector/parsers/typescript"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseFileContent(filename string, content []byte) (ok bool) {
	parser := sitter.NewParser()
	parser.SetLanguage(typescript.GetLanguage())

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		log.Fatal(err)
	}

	root := tree.RootNode()
	templateFilename, err := ExtractTemplateFilename(filename, root, content)
  if err != nil {
    log.Fatal(err)
  }

	log.Printf("Found template for %s at %s", filename, templateFilename)

	return true
}
