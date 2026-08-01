package ast

import (
	"slices"
	"strings"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

// TODO: Should be combined with parser.tcb.Tag
type Tag struct {
	Name       string
	Attributes []string // Has [angular]
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
	return !slices.ContainsFunc(attributes, t.HasAttribute)
}

func (t *Tag) MatchesParsedSelector(selector *Selector) (bool, *Selector) {
	if selector.Tag != "" {
		if t.Name != selector.Tag {
			return false, selector
		}
	}

	if len(selector.Attributes) > 0 {
		if !t.HasAttributes(selector.Attributes) {
			return false, selector
		}
	}

	if len(selector.NotTags) > 0 {
		if slices.Contains(selector.NotTags, t.Name) {
			return false, selector
		}
	}

	if len(selector.NotAttributes) > 0 {
		if !t.NotHasAttributes(selector.NotAttributes) {
			return false, selector
		}
	}

	return true, selector
}

func (t *Tag) MatchesSelector(selector string) (bool, *Selector) {
	s, err := ParseSelector(selector)
	if err != nil {
		return false, s
	}

	return t.MatchesParsedSelector(s)
}

// Deprecated: use ast.ParseSelector
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

			if (child.Type() == "class" || child.Type() == "id") && foundTag.Name == "" {
				foundTag.Name = "div"
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
