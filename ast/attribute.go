package ast

import (
	"ts_inspector/utils"
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
