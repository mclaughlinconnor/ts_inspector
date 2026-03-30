package tcb_cm

import (
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func handleAttribute(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	attribute := Attribute{Name: "", Value: ""}

	a := &attribute
	state.Current.Push((*state.Current.Peek()).Tag.addAttribute(a))

	for i := range node.NamedChildCount() {
		Parse(state, node.NamedChild(int(i)))
	}

	state.Current.Pop()

	return state
}

func handleAttributeName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	(*state.Current.Peek()).Attribute.Name = node.Content(state.Content)

	return state
}

func handleAttributeValue(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	(*state.Current.Peek()).Attribute.Value = node.Content(state.Content)

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
	tag := Tag{Children: HelpfulArray[*Node]{}, Name: ""}
	tagNode := newTagNode(&tag)

	(*state.Current.Peek()).Tag.Children.add(tagNode)
	state.Current.Push(tagNode)

	for i := range node.NamedChildCount() {
		Parse(state, node.NamedChild(int(i)))
	}

	state.Current.Pop()

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

func handleTagName(node *sitter.Node, state *Ast, indexInParent int, internalFuncMap walk.VisitorFuncMap[*Ast]) *Ast {
	(*state.Current.Peek()).Tag.Name = node.Content(state.Content)

	return state
}

func newAttributeNode(attribute *Attribute) *Node {
	return &Node{Kind: KindTag, Attribute: attribute}
}

func newTagNode(tag *Tag) *Node {
	return &Node{Kind: KindTag, Tag: tag}
}
