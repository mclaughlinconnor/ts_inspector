package tcb

import (
	"ts_inspector/parser"

	sitter "github.com/smacker/go-tree-sitter"
)

type Mixin struct {
	renderable
	tcb *Tcb

	Attributes  HelpfulArray[*Node]
	Children    HelpfulArray[*Node]
	Name        string
	NameNode    *sitter.Node
	Node        *sitter.Node
	SourceClass *parser.Class
}

func (m *Mixin) addAttribute(attribute *Attribute) *Node {
	node := newAttributeNode(attribute)

	m.Attributes.add(node)

	return node
}

func (m *Mixin) Render() {
	for _, c := range m.Children.Elements {
		c.Render()
	}
}
