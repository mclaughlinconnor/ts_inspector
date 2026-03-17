package ast

import (
	"strings"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Tag struct {
	Name       string
	Attributes []string
}

func (t *Tag) MatchesSelector(selector string) bool {
	if t.Name == selector {
		return true
	}

	valid, tagName, attrName := ExtractTagNameAndAttrFromSelector(selector)
	if !valid || (tagName != "" && t.Name != tagName) {
		return false
	}

	for _, attr := range t.Attributes {
		if attr == attrName || attr[1:len(attr)-1] == attrName {
			return true
		}
	}

	return false
}

func ExtractTagNameAndAttrFromSelector(selector string) (bool, string, string) {
	firstBracket := strings.Index(selector, "[")
	lastBracket := strings.Index(selector, "]")

	if firstBracket == -1 || lastBracket == -1 {
		return false, "", ""
	}

	tag := selector[:firstBracket]
	attr := selector[firstBracket+1 : lastBracket]

	return true, tag, attr
}

func GetTagNameAtOffset(content string, offset uint32) (string, bool) {
	tagName, _ := utils.ParseFile(false, content, utils.Pug, "", func(root *sitter.Node, content []byte, v string) (string, error) {
		node := HasNodeInHierarchy(root, "tag_name", offset, offset)
		if node == nil {
			return "", nil
		}

		tagName := node.Content([]byte(content))
		return tagName, nil
	})

	if tagName != "" {
		return tagName, true
	}

	return "", false
}

func GetTagAtOffset(content string, offset uint32) (Tag, bool) {
	foundTag := Tag{Name: "", Attributes: []string{}}

	utils.ParseFile(false, content, utils.Pug, "", func(root *sitter.Node, content []byte, v string) (string, error) {
		tag := HasNodeInHierarchy(root, "tag", offset, offset)
		if tag == nil {
			return "", nil
		}

		for i := range tag.NamedChildCount() {
			child := tag.NamedChild(int(i))
			if child == nil {
				continue
			}

			if child.Type() == "tag_name" {
				tagName := child.Content([]byte(content))
				foundTag.Name = tagName
			}

			if child.Type() == "attributes" {
				for j := range child.NamedChildCount() {
					attribute := child.NamedChild(int(j))
					for k := range attribute.NamedChildCount() {
						attributeChild := attribute.NamedChild(int(k))
						if attributeChild.Type() == "attribute_name" {
							foundTag.Attributes = append(foundTag.Attributes, attributeChild.Content(content))
						}
					}
				}
			}
		}

		return "", nil
	})

	if foundTag.Name != "" {
		return foundTag, true
	}

	return Tag{}, false
}
