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
	Errors  []error
	Root    *Node
	Tcb     *Tcb
}

const (
	KindAttribute int = iota
	KindMixin
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
	Mixin     *Mixin
	Tag       *Tag
	Root      *Root
}

type Root struct {
	renderable
	tcb *Tcb

	Children utils.HelpfulArray[*Node]
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
	astVisitorFuncMap["mixin_attributes"] = handleMixinAttributes
	astVisitorFuncMap["mixin_definition"] = handleMixin
	astVisitorFuncMap["mixin_name"] = handleMixinName
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

	var c *utils.HelpfulArray[*Node]

	peek := (*p)

	switch peek.Kind {
	case KindRoot:
		c = &peek.Root.Children
	case KindMixin:
		c = &peek.Mixin.Children
	case KindTag:
		c = &peek.Tag.Children
	default:
		return
	}

	c.Add(n)
}

func (n *Ast) Render() {
	n.Root.Render()
}

func (n *Node) GetChildren() utils.HelpfulArray[*Node] {
	switch n.Kind {
	case KindMixin:
		return n.Mixin.Children
	case KindRoot:
		return n.Root.Children
	case KindTag:
		return n.Tag.Children
	}

	return utils.HelpfulArray[*Node]{}
}

func (n *Node) Render() {
	switch n.Kind {
	case KindMixin:
		n.Mixin.Render()
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

func handleAttribute(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	attribute := Attribute{Name: "", Tag: (*state.Current.Peek()).Tag, tcb: state.Tcb, value: ""}

	a := &attribute
	state.Current.Push((*state.Current.Peek()).Tag.addAttribute(a))

	for i := range node.NamedChildCount() {
		parse(state, node.NamedChild(int(i)))
	}

	state.Current.Pop()

	return state
}

func handleAttributeName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	(*state.Current.Peek()).Attribute.Name = node.Content(state.Content)
	(*state.Current.Peek()).Attribute.NameNode = node

	return state
}

func handleAttributeValue(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	attributeValueNode := node.NamedChild(0)
	if attributeValueNode != nil {
		(*state.Current.Peek()).Attribute.value = attributeValueNode.Content(state.Content)
		(*state.Current.Peek()).Attribute.ValueNode = attributeValueNode
	}

	return state
}

func handleChildNodes(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	prev := state.Current

	for i := range node.NamedChildCount() {
		parse(state, node.NamedChild(int(i)))
	}

	state.Current = prev

	return state
}

func handleMixin(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	mixin := Mixin{Children: utils.HelpfulArray[*Node]{}, Name: "", tcb: state.Tcb}
	mixinNode := newMixinNode(&mixin)

	state.AddChildToCurrent(mixinNode)
	state.Current.Push(mixinNode)

	for i := range node.NamedChildCount() {
		parse(state, node.NamedChild(int(i)))
	}

	state.Current.Pop()

	return state
}

func handleMixinAttributes(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	prev := state.Current

	for i := range node.NamedChildCount() {
		handleMixinAttributeName(node, state, int(i), internalFuncMap)
	}

	state.Current = prev

	return state
}

func handleMixinAttributeName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	attribute := Attribute{Mixin: (*state.Current.Peek()).Mixin, Name: "", tcb: state.Tcb, value: ""}

	a := &attribute
	state.Current.Push((*state.Current.Peek()).Mixin.addAttribute(a))

	(*state.Current.Peek()).Attribute.Name = node.Content(state.Content)
	(*state.Current.Peek()).Attribute.NameNode = node

	state.Current.Pop()

	return state
}

func handleMixinName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	if p := state.Current.Peek(); p != nil {
		(*p).Mixin.Name = node.Content(state.Content)
		(*p).Mixin.NameNode = node
	}

	return state
}

func handleTag(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	tag := Tag{Children: utils.HelpfulArray[*Node]{}, Name: "", tcb: state.Tcb}
	tagNode := newTagNode(&tag)

	state.AddChildToCurrent(tagNode)
	state.Current.Push(tagNode)

	for i := range node.NamedChildCount() {
		parse(state, node.NamedChild(int(i)))
	}

	for _, attr := range tag.Attributes.Elements {
		if strings.HasPrefix(attr.Attribute.Name, "#") {
			valueExpr, err := attr.Attribute.GetExpression()
			if err != nil {
				state.Errors = append(state.Errors, err)
				return state
			}
			ref := TemplateRef{Attribute: attr, Name: attr.Attribute.Name, Tag: &tag, Value: valueExpr.Expression}
			tag.TemplateRefs.Add(ref)
		}
	}

	state.Current.Pop()

	return state
}

func handleTagClass(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	if p := state.Current.Peek(); p != nil && (*p).Tag.Name == "" {
		(*p).Tag.Name = "div"
		(*p).Tag.NameNode = node
	}

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

			(*state.Current.Peek()).Tag.Content.Add(&tagContent)
		}

		return nil, nil
	})

	return state
}

func handleTagId(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	if p := state.Current.Peek(); p != nil && (*p).Tag.Name == "" {
		(*p).Tag.Name = "div"
		(*p).Tag.NameNode = node
	}

	return state
}

func handleTagName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	if p := state.Current.Peek(); p != nil {
		(*p).Tag.Name = node.Content(state.Content)
		(*p).Tag.NameNode = node
	}

	return state
}

func newAttributeNode(attribute *Attribute) *Node {
	return &Node{Kind: KindAttribute, Attribute: attribute}
}

func newMixinNode(mixin *Mixin) *Node {
	return &Node{Kind: KindMixin, Mixin: mixin}
}

func newTagNode(tag *Tag) *Node {
	return &Node{Kind: KindTag, Tag: tag}
}

func parse(state *Ast, root *sitter.Node) {
	walk.VisitNode(root, state, 0, astOptimisedMap, true)
}
