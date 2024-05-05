package main

import (
	"context"
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, filename string, state V) (result V, ok bool)

func HandleTypeScriptFile(filename string) (returnedState State, ok bool) {
	state := NewState()

	return parseFileContent(filename, TypeScript, state,
		parseCallback[State](func(root *sitter.Node, content []byte, filename string, state State) (result State, ok bool) {
			state, err := ExtractTypeScriptUsages(state, root, content)
			if err != nil {
				log.Print(err)
			}

			templateFilename, err := ExtractTemplateFilename(filename, root, content)
			if err != nil {
				log.Fatal(err)
			}

			state, ok = HandlePugFile(templateFilename, state)

			return state, ok
		}))
}

func HandlePugFile(filename string, state State) (returnedState State, ok bool) {
	return parseFileContent(filename, Pug, state,
		parseCallback[State](func(root *sitter.Node, content []byte, filename string, state State) (result State, ok bool) {
			state, err := ExtractPugUsages(state, content)
			if err != nil {
				log.Print(err)
			}
			return state, true
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
