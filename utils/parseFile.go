package utils

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
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
	root, _, err := ParseTextWithTree(content, language)
	if err != nil {
		return nil, err
	}

	return root, nil
}

func ParseTextWithTree(content []byte, language string) (*sitter.Node, *sitter.Tree, error) {
	parser := sitter.NewParser()
	err := parser.SetLanguage(GetLanguage(language))
	if err != nil {
		return nil, nil, err
	}

	tree := parser.Parse(content, nil)
	root := tree.RootNode()

	return root, tree, nil
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
	err = parser.SetLanguage(GetLanguage(language))
	if err != nil {
		return nil, err
	}

	tree := parser.Parse(content, nil)

	return tree.RootNode(), nil
}
