package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type forInStatement struct {
	commonNode
	left  nodeInterface
	right nodeInterface
	body  nodeInterface
}

func (f *forInStatement) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !f.isUnderCursor(offset) {
		return nil, false
	}

	if first && f.getKind() == kind {
		return f, true
	}

	if f.left.isUnderCursor(offset) {
		return f.left.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if f.right.isUnderCursor(offset) {
		return f.right.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if f.body != nil && f.body.isUnderCursor(offset) {
		return f.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || f.getKind() == kind {
		return f, true
	}

	return nil, false
}

func (f *forInStatement) visit(exec func(nodeInterface) int) int {
	if ret := exec(f); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := f.left.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := f.right.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := f.body.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitForInStatement(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	forInStatement := forInStatement{commonNode: makeCommonNode("forInStatement", state, node)}
	forInStatement.setImpl(&forInStatement)

	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return nil, fmt.Errorf("invalid ast: missing left")
	}

	rightNode := node.ChildByFieldName("right")
	if rightNode == nil {
		return nil, fmt.Errorf("invalid ast: missing right")
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, fmt.Errorf("invalid ast: missing body")
	}

	left, err := walk.VisitNode(leftNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	right, err := walk.VisitNode(rightNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	forInStatement.left = left
	forInStatement.right = right
	forInStatement.body = body

	return &forInStatement, nil
}
