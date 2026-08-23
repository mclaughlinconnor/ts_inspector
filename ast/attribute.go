package ast

import (
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func GetAttributeNameAtOffset(content string, offset uint32) (string, bool) {
	c := []byte(content)

	root, err := utils.ParseText(c, utils.Pug)
	if err != nil {
		return "", false
	}

	node := HasNodeInHierarchy(root, "attribute_name", offset, offset)
	if node == nil {
		return "", false
	}

	attributeName := node.Content(c)
	if attributeName == "" {
		return "", false
	}

	return attributeName, true
}

func GetAttributeNameAtOffset2(root *sitter.Node, content string, offset uint32) (string, bool) {
	node := HasNodeInHierarchy(root, "attribute_name", offset, offset)
	if node == nil {
		return "", false
	}

	attributeName := node.Content([]byte(content))
	if attributeName == "" {
		return "", false
	}

	return attributeName, true
}
