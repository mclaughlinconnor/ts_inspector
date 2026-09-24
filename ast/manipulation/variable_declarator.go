package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type variableDeclarator struct {
	commonNode
	name  *identifier
	value nodeInterface
}

func (v *variableDeclarator) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !v.isUnderCursor(offset) {
		return nil, false
	}

	if first && v.getKind() == kind {
		return v, true
	}

	if v.name != nil && v.name.isUnderCursor(offset) {
		return v.name.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if v.value != nil && v.value.isUnderCursor(offset) {
		return v.value.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || v.getKind() == kind {
		return v, true
	}

	return nil, false
}

func (v *variableDeclarator) getNameText() string {
	return v.name.getText()
}

func (v *variableDeclarator) getValueText() string {
	if v.value != nil {
		return v.value.getText()
	}

	return ""
}

func (a *variableDeclarator) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := a.value.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := a.name.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitVariableDeclarator(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	variableDeclarator := variableDeclarator{commonNode: makeCommonNode("variableDeclarator", state, node)}
	variableDeclarator.setImpl(&variableDeclarator)

	valueNode := node.ChildByFieldName("value")

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, fmt.Errorf("invalid ast: missing name")
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

	var name *identifier
	if identifier, isIdentifier := nameCommonNode.isIdentifier(); isIdentifier {
		name = identifier
	} else {
		return nil, fmt.Errorf("invalid ast: identifier isn't an identifier")
	}

	variableDeclarator.value = value
	variableDeclarator.name = name

	return &variableDeclarator, nil
}
