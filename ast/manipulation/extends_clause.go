package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type extendsClause struct {
	commonNode
	typeArguments nodeInterface
	value         nodeInterface
}

func (t *extendsClause) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !t.isUnderCursor(offset) {
		return nil, false
	}

	if first && t.getKind() == kind {
		return t, true
	}

	if t.value != nil && t.value.isUnderCursor(offset) {
		return t.value.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || t.getKind() == kind {
		return t, true
	}

	return nil, false
}

func (t *extendsClause) visit(exec func(nodeInterface) int) int {
	if ret := exec(t); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := t.value.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitExtendsClause(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	extendsClause := extendsClause{commonNode: makeCommonNode("extendsClause", state, node)}
	extendsClause.setImpl(&extendsClause)

	valueNode := node.ChildByFieldName("value")
	if valueNode == nil {
		return nil, fmt.Errorf("invalid ast: missing value")
	}

	value, err := walk.VisitNode(valueNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var typeArguments nodeInterface
	typeArgumentsNode := node.ChildByFieldName("type_arguments")
	if typeArgumentsNode != nil {
		typeArguments, err = walk.VisitNode(typeArgumentsNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	extendsClause.typeArguments = typeArguments
	extendsClause.value = value

	return &extendsClause, nil
}
