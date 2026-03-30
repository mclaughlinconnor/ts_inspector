package tcb_cm

import (
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Ast struct {
	Children HelpfulArray[*Node]
	Content  []byte
	Current  utils.Stack[*Node]
}

const (
	KindAttribute int = iota
	KindTag
)

type renderable interface {
	Render() *string
	Tcb() *Tcb
}

// Node
type Node struct {
	renderable

	Kind int
	Node *sitter.Node

	Attribute *Attribute
	Tag       *Tag
}

type Root struct {
	Children HelpfulArray[*Node]
}

var astOptimisedMap walk.VisitorFuncMap[*Ast]

func InitAstParser() {
	astVisitorFuncMap := walk.NewVisitorFuncsMap[*Ast]()

	astVisitorFuncMap["content"] = handleTagContent
	astVisitorFuncMap["attribute"] = handleAttribute
	astVisitorFuncMap["attribute_name"] = handleAttributeName
	astVisitorFuncMap["attributes"] = handleChildNodes
	astVisitorFuncMap["children"] = handleChildNodes
	astVisitorFuncMap["quoted_attribute_value"] = handleAttributeValue
	astVisitorFuncMap["source_file"] = handleChildNodes
	astVisitorFuncMap["tag"] = handleTag
	astVisitorFuncMap["tag_name"] = handleTagName

	pug := utils.GetLanguage(utils.Pug)
	astOptimisedMap = walk.GenerateSymbolMap(pug, astVisitorFuncMap)
}

func Parse(state *Ast, root *sitter.Node) {
	walk.VisitNode(root, state, 0, astOptimisedMap, true)
}

func (n *Ast) Render() {
	for _, child := range n.Children.Elements {
		child.Render()
	}
}

func (n *Node) Render() {
	switch n.Kind {
	case KindTag:
		n.Tag.Render()
	case KindAttribute:
		n.Attribute.Render()
	}
}
