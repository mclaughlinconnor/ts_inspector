package utils

import (
	"context"
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, v V) (V, error)

func ParseFile[V any](fromDisk bool, source string, language string, v V, callback parseCallback[V]) (V, error) {
	var content []byte
	if fromDisk {
		content = ReadFile(source)
	} else {
		content = []byte(source)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		log.Fatal(err)
	}

	root := tree.RootNode()

	return callback(root, content, v)
}
