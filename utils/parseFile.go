package utils

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func ParseTextFromPath(path string, language string) (*sitter.Node, []byte, error) {
	content, err := ReadFile(path)
	if err != nil {
		return nil, []byte{}, err
	}

	root := ParseText(content, language)

	return root, content, nil
}

func ParseText(content []byte, language string) *sitter.Node {
	root, _ := ParseTextWithTree(content, language)

	return root
}

func ParseTextWithTree(content []byte, language string) (*sitter.Node, *sitter.Tree) {
	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree := parser.Parse(content, nil)
	root := tree.RootNode()

	return root, tree
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

	tree := parser.Parse(content, nil)

	return tree.RootNode(), nil
}
