package utils

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
)

func ParseTextFromPath(path string, language string) (*sitter.Node, []byte, error) {
	content, err := ReadFile(path)
	if err != nil {
		return nil, []byte{}, err
	}

	root, err := ParseText(content, language)
	if err != nil {
		return nil, []byte{}, err
	}

	return root, content, nil
}

func ParseText(content []byte, language string) (*sitter.Node, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		return nil, err
	}

	root := tree.RootNode()

	return root, nil
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
