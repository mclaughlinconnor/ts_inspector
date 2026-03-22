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

	KindTmplAstTemplate
	KindTmplAstVariable
)

// TmplAstNode
type Node struct {
	Tag       *Tag
	Attribute *Attribute
	Variable  *TmplAstVariable
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

type TmplAstVariable struct {
	Name string
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
