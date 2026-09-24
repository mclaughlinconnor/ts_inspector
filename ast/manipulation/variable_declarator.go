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

func (a *variableDeclarator) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	if a.name != nil && a.name.isUnderCursor(offset) {
		return a.name.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.value != nil && a.value.isUnderCursor(offset) {
		return a.value.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
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
	if valueNode == nil {
		return nil, fmt.Errorf("invalid ast: missing value")
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, fmt.Errorf("invalid ast: missing name")
	}

	value, err := walk.VisitNode(valueNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
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
