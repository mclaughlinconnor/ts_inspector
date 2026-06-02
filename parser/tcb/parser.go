package tcb

import (
	"slices"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Ast struct {
	Content []byte
	Current utils.Stack[*Node]
	Root    *Node
	Tcb     *Tcb
}

const (
	KindAttribute int = iota
	KindRoot
	KindTag
)

type renderable interface {
	Render() *string
	Tcb() *Tcb
}

type Node struct {
	renderable

	Kind int
	Node *sitter.Node

	Attribute *Attribute
	Tag       *Tag
	Root      *Root
}

type Root struct {
	renderable
	tcb *Tcb

	Children HelpfulArray[*Node]
}

func (a *Ast) FindTagByTemplateRef(name string) *TemplateRef {
	stack := utils.NewStack[*Node]()
	visited := map[*Node]bool{}

	for _, c := range a.Root.GetChildren().Elements {
		stack.Push(c)
	}

	for !stack.IsEmpty() {
		node := *stack.Pop()

		if visited[node] {
			continue
		}

		visited[node] = true

		if node.Tag != nil {
			index := slices.IndexFunc(node.Tag.TemplateRefs.Elements, func(tr TemplateRef) bool { return strings.TrimPrefix(tr.Name, "#") == name })
			if index != -1 {
				return &node.Tag.TemplateRefs.Elements[index]
			}
		}

		for _, c := range node.GetChildren().Elements {
			if visited[c] {
				continue
			}

			stack.Push(c)
		}
	}

	return nil
}

func (r *Root) Tcb() *Tcb {
	return r.tcb
}

var astOptimisedMap walk.VisitorFuncMap[*Ast]

func initAstParser() {
	astVisitorFuncMap := walk.NewVisitorFuncsMap[*Ast]()

	astVisitorFuncMap["attribute"] = handleAttribute
	astVisitorFuncMap["attribute_name"] = handleAttributeName
	astVisitorFuncMap["attributes"] = handleChildNodes
	astVisitorFuncMap["children"] = handleChildNodes
	astVisitorFuncMap["class"] = handleTagClass
	astVisitorFuncMap["content"] = handleTagContent
	astVisitorFuncMap["id"] = handleTagId
	astVisitorFuncMap["quoted_attribute_value"] = handleAttributeValue
	astVisitorFuncMap["source_file"] = handleChildNodes
	astVisitorFuncMap["tag"] = handleTag
	astVisitorFuncMap["tag_name"] = handleTagName

	pug := utils.GetLanguage(utils.Pug)
	astOptimisedMap = walk.GenerateSymbolMap(pug, astVisitorFuncMap)
}

func Parse(root *sitter.Node, content []byte, tcb *Tcb) *Ast {
	rootAstNode := &Node{Kind: KindRoot, Root: &Root{}}
	state := &Ast{Content: content, Root: rootAstNode, Tcb: tcb}
	tcb.Ast = state
	state.Current.Push(rootAstNode)

	walk.VisitNode(root, state, 0, astOptimisedMap, true)

	return state
}

func (a *Ast) AddChildToCurrent(n *Node) {
	p := a.Current.Peek()
	if p == nil {
		return
	}

	var c *HelpfulArray[*Node]

	peek := (*p)

	switch peek.Kind {
	case KindRoot:
		c = &peek.Root.Children
	case KindTag:
		c = &peek.Tag.Children
	default:
		return
	}

	c.add(n)
}

func (n *Ast) Render() {
	n.Root.Render()
}

func (n *Node) GetChildren() HelpfulArray[*Node] {
	switch n.Kind {
	case KindRoot:
		return n.Root.Children
	case KindTag:
		return n.Tag.Children
	}

	return HelpfulArray[*Node]{}
}

func (n *Node) Render() {
	switch n.Kind {
	case KindRoot:
		for _, c := range n.Root.Children.Elements {
			c.Render()
		}
	case KindTag:
		n.Tag.Render()
	case KindAttribute:
		n.Attribute.Render()
	}
}

func parse(state *Ast, root *sitter.Node) {
	walk.VisitNode(root, state, 0, astOptimisedMap, true)
}
