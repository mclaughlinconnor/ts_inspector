package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type implementsClause struct {
	commonNode
	Implements []nodeInterface
}

func (t *implementsClause) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !t.isUnderCursor(offset) {
		return nil, false
	}

	if first && t.getKind() == kind {
		return t, true
	}

	for _, typeParameter := range t.Implements {
		if typeParameter != nil && typeParameter.isUnderCursor(offset) {
			return typeParameter.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	if kind == NULL_KIND || t.getKind() == kind {
		return t, true
	}

	return nil, false
}

func (t *implementsClause) visit(exec func(nodeInterface) int) int {
	if ret := exec(t); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, typeParameter := range t.Implements {
		if typeParameter == nil {
			continue
		}

		if ret := typeParameter.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitImplementsClause(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	implementsClause := implementsClause{commonNode: makeCommonNode("implementsClause", state, node)}
	implementsClause.setImpl(&implementsClause)

	implements := []nodeInterface{}
	for _, child := range node.NamedChildren(node.Walk()) {
		childCommonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
		if childCommonNode == nil {
			return nil, err
		}

		implements = append(implements, childCommonNode)
	}

	implementsClause.Implements = implements

	return &implementsClause, nil
}
