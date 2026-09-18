package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type arguments struct {
	childedCommonNode
}

func visitArguments(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	arguments := arguments{childedCommonNode: makeCommonChildedNode("arguments", state, node)}
	arguments.setImpl(&arguments)

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

	arguments.children = children

	return &arguments, nil
}
