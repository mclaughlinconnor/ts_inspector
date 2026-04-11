package tcb_cm

import (
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Tag struct {
	tcb *Tcb

	Attributes  HelpfulArray[*Node]
	Children    HelpfulArray[*Node]
	Content     HelpfulArray[*TagContent]
	Name        string
	NameNode    *sitter.Node
	Node        *sitter.Node
	SourceClass *parser.Class
}

type TagContent struct {
	Interpolation string
	Node          *sitter.Node
	Text          string
}

type TagContentArray struct {
	elems []*TagContent
}

func (t *Tag) addAttribute(attribute *Attribute) *Node {
	node := newAttributeNode(attribute)

	t.Attributes.add(node)

	return node
}

func (t *Tag) matchesSelector(selector string) bool {
	if t.Name == selector {
		return true
	}

	valid, tagName, attrName := ast.ExtractTagNameAndAttrFromSelector(selector)
	if !valid || (tagName != "" && t.Name != tagName) {
		return false
	}

	for _, attr := range t.Attributes.Elements {
		attr := attr.Attribute.Name

		if attr == attrName {
			return true
		}

		angularlessAttr, _ := utils.StripAngularFromAttribute(attr)
		if angularlessAttr == attrName {
			return true
		}
	}

	return false
}

func (t *Tag) Render() {
	tcb := t.Tcb()

	value := StatementParts{}
	value.AddVirtPart("document.createElement(\"")
	value.AddVirtPart(t.Name)
	value.AddVirtPart("\")")

	tcb.CreateVar(value)

	for _, a := range t.Attributes.Elements {
		a.Render()
	}

	for _, c := range t.Children.Elements {
		c.Render()
	}
}

func (t *Tag) Tcb() *Tcb {
	return t.tcb
}
