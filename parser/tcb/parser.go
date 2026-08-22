package tcb

import (
	"errors"
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

func Parse(root *sitter.Node, content []byte, tcb *Tcb) (*Ast, error) {
	rootAstNode := &Node{Kind: KindRoot, Root: &Root{}}
	state := &Ast{Content: content, Root: rootAstNode, Tcb: tcb}
	tcb.Ast = state
	state.Current.Push(rootAstNode)

	_, err := walk.VisitNode(root, state, 0, astOptimisedMap, true)
	if err != nil {
		return nil, err
	}

	return state, err
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

func (n *Ast) Render() error {
	return n.Root.Render()
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

func (n *Node) Render() error {
	var err error

	switch n.Kind {
	case KindMixin:
		err = n.Mixin.Render()
	case KindRoot:
		for _, c := range n.Root.Children.Elements {
			err = c.Render()
			if err != nil {
				continue
			}
		}
	case KindTag:
		err = n.Tag.Render()
	case KindAttribute:
		n.Attribute.Render()
	}

	return err
}

func handleAttribute(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	attribute := Attribute{Name: "", Node: node, Tag: (*state.Current.Peek()).Tag, tcb: state.Tcb, value: ""}

	a := &attribute
	state.Current.Push((*state.Current.Peek()).Tag.addAttribute(a))

	for i := range node.NamedChildCount() {
		err := parse(state, node.NamedChild(int(i)))
		if err != nil {
			return nil, err
		}
	}

	state.Current.Pop()

	return state, nil
}

func handleAttributeName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	(*state.Current.Peek()).Attribute.Name = node.Content(state.Content)
	(*state.Current.Peek()).Attribute.NameNode = node

	return state, nil
}

func handleAttributeValue(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	attributeValueNode := node.NamedChild(0)
	if attributeValueNode != nil {
		(*state.Current.Peek()).Attribute.value = attributeValueNode.Content(state.Content)
		(*state.Current.Peek()).Attribute.ValueNode = attributeValueNode
	}

	return state, nil
}

func handleChildNodes(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	prev := state.Current

	for i := range node.NamedChildCount() {
		err := parse(state, node.NamedChild(int(i)))
		if err != nil {
			return nil, err
		}
	}

	state.Current = prev

	return state, nil
}

func handleMixin(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	mixin := Mixin{Children: utils.HelpfulArray[*Node]{}, Name: "", Node: node, tcb: state.Tcb}
	mixinNode := newMixinNode(&mixin)

	state.AddChildToCurrent(mixinNode)
	state.Current.Push(mixinNode)

	for i := range node.NamedChildCount() {
		err := parse(state, node.NamedChild(int(i)))
		if err != nil {
			return nil, err
		}
	}

	state.Current.Pop()

	return state, nil
}

func handleMixinAttributes(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	prev := state.Current

	for i := range node.NamedChildCount() {
		handleMixinAttributeName(node, state, int(i), internalFuncMap)
	}

	state.Current = prev

	return state, nil
}

func handleMixinAttributeName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	attribute := Attribute{Mixin: (*state.Current.Peek()).Mixin, Name: "", Node: node, tcb: state.Tcb, value: ""}

	a := &attribute
	state.Current.Push((*state.Current.Peek()).Mixin.addAttribute(a))

	(*state.Current.Peek()).Attribute.Name = node.Content(state.Content)
	(*state.Current.Peek()).Attribute.NameNode = node

	state.Current.Pop()

	return state
}

func handleMixinName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	if p := state.Current.Peek(); p != nil {
		(*p).Mixin.Name = node.Content(state.Content)
		(*p).Mixin.NameNode = node
	}

	return state, nil
}

func handleTag(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	tag := Tag{Children: utils.HelpfulArray[*Node]{}, Name: "", Node: node, tcb: state.Tcb}
	tagNode := newTagNode(&tag)

	state.AddChildToCurrent(tagNode)
	state.Current.Push(tagNode)

	for i := range node.NamedChildCount() {
		err := parse(state, node.NamedChild(int(i)))
		if err != nil {
			return nil, err
		}
	}

	for _, attr := range tag.Attributes.Elements {
		if strings.HasPrefix(attr.Attribute.Name, "#") {
			valueExpr, err := attr.Attribute.GetExpression()
			if err != nil {
				return state, err
			}

			if valueExpr == nil {
				return state, errors.New("no attribute expression")
			}

			ref := TemplateRef{Attribute: attr, Name: attr.Attribute.Name, Tag: &tag, Value: valueExpr.Expression}
			tag.TemplateRefs.Add(ref)
		}
	}

	state.Current.Pop()

	return state, nil
}

func handleTagClass(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	if p := state.Current.Peek(); p != nil && (*p).Tag.Name == "" {
		(*p).Tag.Name = "div"
		(*p).Tag.NameNode = node
	}

	return state, nil
}

func handleTagContent(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	content := []byte(node.Content(state.Content))
	root, err := utils.ParseText([]byte(node.Content(state.Content)), utils.AngularContent)

	if err != nil {
		return state, err
	}

	for i := range root.ChildCount() {
		child := root.Child(int(i))

		tagContent := TagContent{}
		switch child.Type() {
		case "text":
			tagContent.Text = child.Content(content)
		case "interpolation":
			// TODO: also parse the interpolation content
			// tagContent.Interpolation = child.Content(content)
			//
			// interpolationContentWithBraces := child.Content(content)
			// interpolationContent := interpolationContentWithBraces[2 : len(interpolationContentWithBraces)-2]
			//
			// _, _ = utils.ParseText2([]byte(interpolationContent), utils.AngularExpr)
		}

		(*state.Current.Peek()).Tag.Content.Add(&tagContent)
	}

	return state, nil
}

func handleTagId(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	if p := state.Current.Peek(); p != nil && (*p).Tag.Name == "" {
		(*p).Tag.Name = "div"
		(*p).Tag.NameNode = node
	}

	return state, nil
}

func handleTagName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) (*Ast, error) {
	if p := state.Current.Peek(); p != nil {
		(*p).Tag.Name = node.Content(state.Content)
		(*p).Tag.NameNode = node
	}

	return state, nil
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

func parse(state *Ast, root *sitter.Node) error {
	_, err := walk.VisitNode(root, state, 0, astOptimisedMap, true)
	if err != nil {
		return err
	}

	return nil
}
