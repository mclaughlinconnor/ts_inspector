package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type unhandled struct {
	childedCommonNode
}

func (u *unhandled) invert() {
	for _, child := range u.getChildren() {
		invertableChild, isInvertable := child.isInvertable()
		if isInvertable {
			invertableChild.invert()
		}
	}
}

func visitUnhandled(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	unhandled := unhandled{childedCommonNode: makeCommonChildedNode("unhandled", state, node)}
	unhandled._self = &unhandled

	children := []nodeInterface{}

	for i := range node.NamedChildCount() {
		child, err := walk.VisitNode(node.NamedChild(i), state, i, funcMap, false)
		if err != nil {
			return nil, err
		}

		if child.getId() == state.getId() {
			continue
		}

		children = append(children, child)
	}

	unhandled.children = children

	return &unhandled, nil
}
