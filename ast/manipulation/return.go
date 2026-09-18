package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type returnExpression struct {
	childedCommonNode
	expression nodeInterface
}

func visitReturn(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	returnExpression := returnExpression{childedCommonNode: makeCommonChildedNode("returnExpression", state, node)}
	returnExpression.setImpl(&returnExpression)

	children := []nodeInterface{}
	var expression nodeInterface = nil

	for i := range node.NamedChildCount() {
		childNode := node.NamedChild(i)
		child, err := walk.VisitNode(childNode, state, i, funcMap, false)
		if err != nil {
			return nil, err
		}

		// VisitNode returns its state if nothing is visited. Don't create cycles in the tree
		if child.getId() == state.getId() {
			continue
		}

		children = append(children, child)

		if childNode.Kind() != "comment" {
			expression = child
		}
	}

	returnExpression.children = children
	returnExpression.expression = expression

	return &returnExpression, nil
}
