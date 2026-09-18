package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type awaitExpression struct {
	childedCommonNode
	expression nodeInterface
}

func (e *awaitExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(e); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := e.expression.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitAwaitExpression(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	awaitExpression := awaitExpression{childedCommonNode: makeCommonChildedNode("awaitExpression", state, node)}
	awaitExpression.setImpl(&awaitExpression)

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

	awaitExpression.children = children
	awaitExpression.expression = expression

	return &awaitExpression, nil
}
