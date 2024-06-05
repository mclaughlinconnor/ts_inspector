package utils

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, v V) (V, error)

func ParseFile[V any](fromDisk bool, source string, language string, v V, callback parseCallback[V]) (V, error) {
	var content []byte
	var err error
	if fromDisk {
		content, err = ReadFile(source)
		if err != nil {
			return v, err
		}
	} else {
		content = []byte(source)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		return v, err
	}

	root := tree.RootNode()

	return callback(root, content, v)
}

func ParseText[V any](content []byte, language string, v V, callback parseCallback[V]) (V, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		return v, err
	}

	root := tree.RootNode()

	return callback(root, content, v)
}

func GetRootNode(fromDisk bool, source string, language string) (*sitter.Node, error) {
	var content []byte
	var err error
	if fromDisk {
		content, err = ReadFile(source)
		if err != nil {
			return nil, err
		}
	} else {
		content = []byte(source)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		return nil, err
	}

	return tree.RootNode(), nil
}
