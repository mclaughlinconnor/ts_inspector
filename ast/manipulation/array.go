package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type array struct {
	childedCommonNode
}

func visitArray(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	array := array{childedCommonNode: makeCommonChildedNode("array", state, node)}
	array.setImpl(&array)

	children := []nodeInterface{}

	for i := range node.NamedChildCount() {
		child, err := walk.VisitNode(node.NamedChild(i), state, i, funcMap, false)
		if err != nil {
			return nil, err
		}

		// VisitNode returns its state if nothing is visited. Don't create cycles in the tree
		if child.getId() == state.getId() {
			continue
		}

		children = append(children, child)
	}

	array.children = children

	return &array, nil
}
