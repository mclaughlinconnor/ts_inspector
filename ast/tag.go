package ast

import (
	"slices"
	"strings"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Tag struct {
	Name       string
	Attributes []string
}

func (t *Tag) HasAttribute(attribute string) bool {
	strippedAttr, _ := utils.StripAngularFromAttribute(attribute)
	return slices.ContainsFunc(t.Attributes, func(a string) bool { sa, _ := utils.StripAngularFromAttribute(a); return sa == strippedAttr })
}

func (t *Tag) HasAttributes(attributes []string) bool {
	for _, attribute := range attributes {
		if !t.HasAttribute(attribute) {
			return false
		}
	}

	return true
}

func (t *Tag) NotHasAttributes(attributes []string) bool {
	for _, attribute := range attributes {
		if t.HasAttribute(attribute) {
			return false
		}
	}

	return true
}

func (t *Tag) MatchesSelector(selector string) (bool, *Selector) {
	s, err := ParseSelector(selector)
	if err != nil {
		return false, s
	}

	if s.Tag != "" {
		if t.Name != s.Tag {
			return false, s
		}
	}

	if len(s.Attributes) > 0 {
		if !t.HasAttributes(s.Attributes) {
			return false, s
		}
	}

	if len(s.NotTags) > 0 {
		for _, tag := range s.NotTags {
			if t.Name == tag {
				return false, s
			}
		}
	}

	if len(s.NotAttributes) > 0 {
		if !t.NotHasAttributes(s.NotAttributes) {
			return false, s
		}
	}

	return true, s
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
