package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type continueExpression struct {
	commonNode
	label nodeInterface
}

func (i *continueExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
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

func (e *continueExpression) visit(exec func(nodeInterface) int) int {
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

func visitContinue(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	continueExpression := continueExpression{commonNode: makeCommonNode("continueExpression", state, node)}
	continueExpression.setImpl(&continueExpression)

	labelNode := node.ChildByFieldName("label")
	if labelNode == nil {
		return nil, fmt.Errorf("invalid ast: missing label")
	}

	label, err := walk.VisitNode(labelNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	continueExpression.label = label

	return &continueExpression, nil
}
