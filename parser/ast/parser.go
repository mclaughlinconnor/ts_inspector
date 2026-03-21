package ast

import (
	"ts_inspector/ast"
	"ts_inspector/ast/walk"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Ast struct {
	Children HelpfulArray[*Node]
	Content  []byte
	Current  utils.Stack[*Node]
}

const (
	KindRoot int = iota
	KindTag
	KindAttribute
)

type Node struct {
	Tag       *Tag
	Attribute *Attribute
	Variable  *Variable
	Kind      int
}

type Attribute struct {
	Name        string
	SourceClass *parser.Class
	Value       string
}

type HelpfulArray[T any] struct {
	elems []T
}

type Root struct {
	Children HelpfulArray[*Node]
}

type Tag struct {
	Attributes  HelpfulArray[*Node]
	Content     HelpfulArray[*TagContent]
	Children    HelpfulArray[*Node]
	Name        string
	SourceClass *parser.Class
}

type TagContent struct {
	Text          string
	Interpolation string
}

type TagContentArray struct {
	elems []*TagContent
}

type Variable struct {
	Name string
}

var visitorFuncMap = walk.NewVisitorFuncsMap[*Ast]()
var optimisedMap walk.VisitorFuncMap[*Ast]

func InitAstParser() {
	visitorFuncMap["content"] = handleTagContent
	visitorFuncMap["attribute"] = handleAttribute
	visitorFuncMap["attribute_name"] = handleAttributeName
	visitorFuncMap["attributes"] = handleChildNodes
	visitorFuncMap["children"] = handleChildNodes
	visitorFuncMap["quoted_attribute_value"] = handleAttributeValue
	visitorFuncMap["source_file"] = handleChildNodes
	visitorFuncMap["tag"] = handleTag
	visitorFuncMap["tag_name"] = handleTagName

	pug := utils.GetLanguage(utils.Pug)
	optimisedMap = walk.GenerateSymbolMap(pug, visitorFuncMap)
}

func Parse(state *Ast, root *sitter.Node) {
	walk.VisitNode(root, state, 0, optimisedMap, true)
}

func (h *HelpfulArray[T]) add(elem T) {
	h.elems = append(h.elems, elem)
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

	for _, attr := range t.Attributes.elems {
		attr := attr.Attribute.Name
		if attr == attrName || attr[1:len(attr)-1] == attrName {
			return true
		}
	}

	return false
}
