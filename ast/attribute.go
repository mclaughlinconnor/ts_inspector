package ast

import (
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func GetAttributeNameAtOffset(content string, offset uint32) (string, bool) {
	attributeName, _ := utils.ParseFile(false, content, utils.Pug, "", func(root *sitter.Node, content []byte, v string) (string, error) {
		node := HasNodeInHierarchy(root, "attribute_name", offset, offset)
		if node == nil {
			return "", nil
		}

		tagName := node.Content([]byte(content))
		return tagName, nil
	})

	if attributeName != "" {
		return attributeName, true
	}

	return "", false
}
