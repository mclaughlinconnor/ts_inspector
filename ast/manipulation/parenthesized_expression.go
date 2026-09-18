package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type parenthesizedExpression struct {
	commonNode
	child nodeInterface
}

func (p *parenthesizedExpression) getNode() impl[nodeInterface] {
	commonNode, _ := p.child.(impl[nodeInterface])
	return commonNode
}

func visitParenthesizedExpression(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	parenthsizedExpression := parenthesizedExpression{commonNode: makeCommonNode("parenthesizedExpression", state, node)}
	parenthsizedExpression._self = &parenthsizedExpression

	if node.NamedChildCount() == 0 {
		return &parenthsizedExpression, nil
	}

	if node.NamedChildCount() > 1 {
		return nil, fmt.Errorf("parenthesized_expression has more than one child")
	}

	child, err := walk.VisitNode(node.NamedChild(0), state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	parenthsizedExpression.child = child

	return &parenthsizedExpression, nil
}
