package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type VariableDeclaration struct {
	commonNode
	Declarator *VariableDeclarator
	IsExport   bool
}

func (a *VariableDeclaration) _buildCfgBlock() (bool, error) {
	a.getCfg().addInstruction(instructionAssign, a.Declarator.getNameText(), a, a.Declarator.getValueText())

	return true, nil
}

func (a *VariableDeclaration) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	if a.Declarator != nil && a.Declarator.isUnderCursor(offset) {
		return a.Declarator.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (a *VariableDeclaration) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := a.Declarator.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitVariableDeclaration(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	variableDeclaration := VariableDeclaration{commonNode: makeCommonNode("variableDeclaration", state, node)}
	variableDeclaration.setImpl(&variableDeclaration)

	declaratorNode := node.NamedChild(0)
	if declaratorNode == nil {
		return nil, fmt.Errorf("invalid ast: missing declarator")
	}

	declaratorCommonNode, err := walk.VisitNode(declaratorNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var declarator *VariableDeclarator
	if variableDeclarator, isVariableDeclarator := IsNode[*VariableDeclarator](declaratorCommonNode); isVariableDeclarator {
		declarator = variableDeclarator
	} else {
		return nil, fmt.Errorf("invalid ast: declarator is not a variable_declarator")
	}

	variableDeclaration.Declarator = declarator
	_, variableDeclaration.IsExport = IsNode[*ExportStatement](state)

	return &variableDeclaration, nil
}
