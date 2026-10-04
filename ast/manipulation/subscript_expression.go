package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type subscriptExpression struct {
	commonMemberSubscript
	index nodeInterface
}

func (f *subscriptExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !f.isUnderCursor(offset) {
		return nil, false
	}

	if first && f.getKind() == kind {
		return f, true
	}

	if f.object != nil && f.object.isUnderCursor(offset) {
		return f.object.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if f.index != nil && f.index.isUnderCursor(offset) {
		return f.index.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || f.getKind() == kind {
		return f, true
	}

	return nil, false
}

func (f *subscriptExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(f); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := f.object.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := f.index.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitSubscriptExpression(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	commonMemberSubscript, err := visitCommonMemberSubscript(node, state, 0, funcMap)
	if err != nil {
		return nil, err
	}

	if commonMemberSubscript == nil {
		return nil, newAstError(commonMemberSubscript, "missing common member subscript")
	}

	subscriptExpression := subscriptExpression{commonMemberSubscript: *commonMemberSubscript}
	subscriptExpression.kind = "subscriptExpression"
	subscriptExpression.setImpl(&subscriptExpression)

	indexNode := node.ChildByFieldName("index")
	if indexNode == nil {
		return nil, newAstErrorTS(node, state, "missing index")
	}

	index, err := walk.VisitNode(indexNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	subscriptExpression.index = index

	return &subscriptExpression, nil
}
