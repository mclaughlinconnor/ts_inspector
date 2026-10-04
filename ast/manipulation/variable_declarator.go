package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type VariableDeclarator struct {
	commonNode
	Name  *Identifier
	Value nodeInterface
}

func (v *VariableDeclarator) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !v.isUnderCursor(offset) {
		return nil, false
	}

	if first && v.getKind() == kind {
		return v, true
	}

	if v.Name != nil && v.Name.isUnderCursor(offset) {
		return v.Name.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if v.Value != nil && v.Value.isUnderCursor(offset) {
		return v.Value.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || v.getKind() == kind {
		return v, true
	}

	return nil, false
}

func (v *VariableDeclarator) getNameText() string {
	return v.Name.getText()
}

func (v *VariableDeclarator) getValueText() string {
	if v.Value != nil {
		return v.Value.getText()
	}

	return ""
}

func (a *VariableDeclarator) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := a.Value.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := a.Name.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitVariableDeclarator(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	variableDeclarator := VariableDeclarator{commonNode: makeCommonNode("variableDeclarator", state, node)}
	variableDeclarator.setImpl(&variableDeclarator)

	valueNode := node.ChildByFieldName("value")

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, newAstErrorTS(node, state, "missing name")
	}

	var value nodeInterface
	var err error

	if valueNode != nil {
		value, err = walk.VisitNode(valueNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	nameCommonNode, err := walk.VisitNode(nameNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var name *Identifier
	if identifier, isIdentifier := IsNode[*Identifier](nameCommonNode); isIdentifier {
		name = identifier
	} else {
		return nil, newAstError(identifier, "identifier isn't an identifier: "+identifier.getKind())
	}

	variableDeclarator.Value = value
	variableDeclarator.Name = name

	return &variableDeclarator, nil
}
