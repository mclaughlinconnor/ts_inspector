package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type breakExpression struct {
	commonNode
	label nodeInterface
}

func (i *breakExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !i.isUnderCursor(offset) {
		return nil, false
	}

	if first && i.getKind() == kind {
		return i, true
	}

	if i.label.isUnderCursor(offset) {
		return i.label.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || i.getKind() == kind {
		return i, true
	}

	return nil, false
}

func (e *breakExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(e); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := e.label.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitBreak(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	breakExpression := breakExpression{commonNode: makeCommonNode("breakExpression", state, node)}
	breakExpression.setImpl(&breakExpression)

	labelNode := node.ChildByFieldName("label")
	if labelNode == nil {
		return nil, fmt.Errorf("invalid ast: missing label")
	}

	label, err := walk.VisitNode(labelNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	breakExpression.label = label

	return &breakExpression, nil
}
