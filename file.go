package main

import (
	"context"
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, filename string, state V) (result V, ok bool)

func HandleTypeScriptFile(filename string) (returnedState Usages, ok bool) {
	state := Usages{}

	return parseFileContent(filename, TypeScript, state,
		parseCallback[Usages](func(root *sitter.Node, content []byte, filename string, state Usages) (result Usages, ok bool) {
			usages := Usages{}
			usages, err := ExtractTypeScriptUsages(usages, root, content)
			if err != nil {
				log.Print(err)
			}

			templateFilename, err := ExtractTemplateFilename(filename, root, content)
			if err != nil {
				log.Fatal(err)
			}

			HandlePugFile(templateFilename, state)

			return nil, true
		}))
}

func HandlePugFile(filename string, state Usages) (returnedState Usages, ok bool) {
	return parseFileContent(filename, Pug, state,
		parseCallback[Usages](func(root *sitter.Node, content []byte, filename string, usages Usages) (result Usages, ok bool) {
			usages, err := ExtractPugUsages(usages, content)
			if err != nil {
				log.Print(err)
			}
			return nil, true
		}))
}

func parseFileContent[V any](filename string, language string, state V, callback parseCallback[V]) (result V, ok bool) {
	content := ReadFile(filename)

	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		log.Fatal(err)
	}

	root := tree.RootNode()

	return callback(root, content, filename, state)
}
