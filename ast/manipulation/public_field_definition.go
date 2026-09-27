package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type publicFieldDefinition struct {
	commonField
	ttype nodeInterface
}

func (a *publicFieldDefinition) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	under, found := a.commonField.getAstNodeOfKindAtOffset(offset, kind, first)
	if found && under != a {
		return under, found
	}

	if a.ttype != nil && a.ttype.isUnderCursor(offset) {
		return a.ttype.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (a *publicFieldDefinition) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if a.accessibility != nil {
		if ret := a.accessibility.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := a.name.visit(exec); ret == VisitAbort {
		return ret
	}

	if a.ttype != nil {
		if ret := a.ttype.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitPublicFieldDefinition(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	commonFieldCommonNode, err := visitCommonField(node, state, 0, funcMap)
	if err != nil {
		return nil, err
	}

	if commonFieldCommonNode == nil {
		return nil, fmt.Errorf("invalid ast: missing common field")
	}

	commonField, isCommonField := IsNode[*commonField](commonFieldCommonNode)
	if !isCommonField {
		return nil, fmt.Errorf("invalid ast: common field is't a common field")
	}

	publicFieldDefinition := publicFieldDefinition{commonField: *commonField}
	publicFieldDefinition.kind = "publicFieldDefinition"
	publicFieldDefinition.setImpl(&publicFieldDefinition)

	typeNode := node.ChildByFieldName("type")

	var ttype nodeInterface
	if typeNode != nil {
		ttype, err = walk.VisitNode(typeNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	publicFieldDefinition.ttype = ttype

	return &publicFieldDefinition, nil
}
