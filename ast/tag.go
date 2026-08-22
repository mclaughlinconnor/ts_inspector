package ast

import (
	"slices"
	"strings"
	"ts_inspector/utils"
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

func (t *Tag) MatchesSelector(selector string) (bool, *Selector, error) {
	s, err := ParseSelector(selector)
	if err != nil {
		return false, nil, err
	}

	matches, s := t.MatchesParsedSelector(s)

	return matches, s, nil
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
	c := []byte(content)

	root, err := utils.ParseText(c, utils.Pug)
	if err != nil {
		return "", false
	}

	node := HasNodeInHierarchy(root, "tag_name", offset, offset)
	if node == nil {
		return "", false
	}

	tagName := node.Content(c)

	if tagName == "" {
		return "", false
	}

	return tagName, true
}

func GetTagAtOffset(content string, offset uint32) (Tag, bool) {
	c := []byte(content)

	foundTag := Tag{Name: "", Attributes: []string{}}

	root, err := utils.ParseText(c, utils.Pug)
	if err != nil {
		return foundTag, false
	}

	tag := HasNodeInHierarchy(root, "tag", offset, offset)
	if tag == nil {
		return foundTag, false
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
						foundTag.Attributes = append(foundTag.Attributes, attributeChild.Content(c))
					}
				}
			}
		}
	}

	if foundTag.Name != "" {
		return foundTag, true
	}

	return Tag{}, false
}

func GetTagAtOffset2(content string, offset uint32) (*Tag, bool) {
	c := []byte(content)

	root, err := utils.ParseText(c, utils.Pug)
	if err != nil {
		return nil, false
	}

	tag := HasNodeInHierarchy(root, "tag", offset, offset)
	if tag == nil {
		return nil, false
	}

	cursorOnTagName := false
	foundTag := Tag{Name: "", Attributes: []string{}}

	for i := range tag.NamedChildCount() {
		child := tag.NamedChild(int(i))
		if child == nil {
			continue
		}

		if child.Type() == "tag_name" {
			cursorOnTagName = cursorOnTagName || (child.StartByte() <= offset && offset <= child.EndByte())
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
						foundTag.Attributes = append(foundTag.Attributes, attributeChild.Content(c))
					}
				}
			}
		}
	}

	if foundTag.Name == "" {
		return nil, false
	}

	return &foundTag, cursorOnTagName
}
