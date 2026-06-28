package tcb

import (
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func handleAttribute(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	attribute := Attribute{Name: "", Tag: (*state.Current.Peek()).Tag, Value: "", tcb: state.Tcb}

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
		(*state.Current.Peek()).Attribute.Value = attributeValueNode.Content(state.Content)
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
	mixin := Mixin{Children: HelpfulArray[*Node]{}, Name: "", tcb: state.Tcb}
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
	attribute := Attribute{Name: "", Mixin: (*state.Current.Peek()).Mixin, Value: "", tcb: state.Tcb}

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
	tag := Tag{Children: HelpfulArray[*Node]{}, Name: "", tcb: state.Tcb}
	tagNode := newTagNode(&tag)

	state.AddChildToCurrent(tagNode)
	state.Current.Push(tagNode)

	for i := range node.NamedChildCount() {
		parse(state, node.NamedChild(int(i)))
	}

	for _, attr := range tag.Attributes.Elements {
		if strings.HasPrefix(attr.Attribute.Name, "#") {
			ref := TemplateRef{Attribute: attr, Name: attr.Attribute.Name, Tag: &tag, Value: attr.Attribute.Value}
			tag.TemplateRefs.add(ref)
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

			(*state.Current.Peek()).Tag.Content.add(&tagContent)
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
