package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type memberExpression struct {
	commonMemberSubscript
	property *propertyIdentifier
}

func (f *memberExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !f.isUnderCursor(offset) {
		return nil, false
	}

	if first && f.getKind() == kind {
		return f, true
	}

	if f.object != nil && f.object.isUnderCursor(offset) {
		return f.object.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if f.property != nil && f.property.isUnderCursor(offset) {
		return f.property.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || f.getKind() == kind {
		return f, true
	}

	return nil, false
}

func (f *memberExpression) visit(exec func(nodeInterface) int) int {
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

	if ret := f.property.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitMemberExpression(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	commonMemberSubscript, err := visitCommonMemberSubscript(node, state, 0, funcMap)
	if err != nil {
		return nil, err
	}

	if commonMemberSubscript == nil {
		return nil, newAstError(commonMemberSubscript, "missing common member subscript")
	}

	memberExpression := memberExpression{commonMemberSubscript: *commonMemberSubscript}
	memberExpression.kind = "memberExpression"
	memberExpression.setImpl(&memberExpression)

	propertyNode := node.ChildByFieldName("property")
	if propertyNode == nil {
		return nil, newAstErrorTS(node, state, "missing property")
	}

	propertyCommonNode, err := walk.VisitNode(propertyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var property *propertyIdentifier
	if propertyIdentifier, isPropertyIdentifier := IsNode[*propertyIdentifier](propertyCommonNode); isPropertyIdentifier {
		property = propertyIdentifier
	} else {
		return nil, newAstError(propertyIdentifier, "property isn't a property: "+propertyIdentifier.getKind())
	}

	memberExpression.property = property

	return &memberExpression, nil
}
