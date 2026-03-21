package ast

import (
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Ast struct {
	Children []*Node
	Content  []byte
	Current  *Node
}

const (
	KindRoot int = iota
	KindTag
	KindAttribute
)

type Node struct {
	Attributes []*Node
	Content    []*TagContent
	Children   []*Node
	Kind       int
	Name       string
	Value      string
}

type TagContent struct {
	Text          string
	Interpolation string
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

func handleAttribute(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	attribute := Node{Kind: KindAttribute, Name: "", Value: ""}

	prev := state.Current

	ap := &attribute

	state.Current.Attributes = append(state.Current.Attributes, ap)
	state.Current = ap

	for i := range node.NamedChildCount() {
		Parse(state, node.NamedChild(int(i)))
	}

	state.Current = prev

	return state
}

func handleAttributeName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	state.Current.Name = node.Content(state.Content)

	return state
}

func handleAttributeValue(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	state.Current.Value = node.Content(state.Content)

	return state
}

func handleChildNodes(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	prev := state.Current

	for i := range node.NamedChildCount() {
		Parse(state, node.NamedChild(int(i)))
	}

	state.Current = prev

	return state
}

func handleTag(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	tag := Node{Kind: KindTag, Children: []*Node{}, Name: ""}

	prev := state.Current

	tp := &tag

	state.Current.Children = append(state.Current.Children, tp)
	state.Current = tp

	for i := range node.NamedChildCount() {
		Parse(state, node.NamedChild(int(i)))
	}

	state.Current = prev

	return state
}

func handleTagContent(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	utils.ParseText([]byte(node.Content(state.Content)), utils.AngularContent, nil, func(root *sitter.Node, content []byte, _ *sitter.Node) (*sitter.Node, error) {
		for i := range root.ChildCount() {
			child := root.Child(int(i))

			tagContent := TagContent{}
			switch child.Type() {
			case "text":
				tagContent.Text = child.Content(content)
			case "interpolation":
				tagContent.Interpolation = child.Content(content)

				interpolationContentWithBraces := child.Content(content)
				interpolationContent := interpolationContentWithBraces[2 : len(interpolationContentWithBraces)-2]

				utils.ParseText([]byte(interpolationContent), utils.AngularExpr, nil, func(root *sitter.Node, content []byte, _ *sitter.Node) (*sitter.Node, error) {
					// TODO: also parse the interpolation content
					
					return nil, nil
				})
			}

			state.Current.Content = append(state.Current.Content, &tagContent)
		}

		return nil, nil
	})

	state.Current.Name = node.Content(state.Content)

	return state
}

func handleTagName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	state.Current.Name = node.Content(state.Content)

	return state
}
