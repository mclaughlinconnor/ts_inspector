package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type lexicalDeclarationKind string

type lexicalDeclarationKindStruct struct {
	CONST lexicalDeclarationKind
	LET   lexicalDeclarationKind
}

var lexicalDeclarationKindEnum = lexicalDeclarationKindStruct{CONST: "const", LET: "let"}

type lexicalDeclaration struct {
	commonNode
	declarator *variableDeclarator
	kind       string
}

func (l *lexicalDeclaration) _buildCfgBlock() (bool, error) {
	l.getCfg().addInstruction(instructionAssign, l.declarator.getNameText(), l, l.declarator.getValueText())

	return true, nil
}

func (l *lexicalDeclaration) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !l.isUnderCursor(offset) {
		return nil, false
	}

	if first && l.getKind() == kind {
		return l, true
	}

	if l.declarator != nil && l.declarator.isUnderCursor(offset) {
		return l.declarator.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || l.getKind() == kind {
		return l, true
	}

	return nil, false
}

func (l *lexicalDeclaration) visit(exec func(nodeInterface) int) int {
	if ret := exec(l); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := l.declarator.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitlexicalDeclaration(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	lexicalDeclaration := lexicalDeclaration{commonNode: makeCommonNode("lexicalDeclaration", state, node)}
	lexicalDeclaration.setImpl(&lexicalDeclaration)

	kindNode := node.ChildByFieldName("kind")
	if kindNode == nil {
		return nil, fmt.Errorf("invalid ast: missing kind")
	}

	declaratorNode := node.NamedChild(0)
	if declaratorNode == nil {
		return nil, fmt.Errorf("invalid ast: missing declarator")
	}

	kind := kindNode.Utf8Text([]byte(lexicalDeclaration.getProgramText()))

	declaratorCommonNode, err := walk.VisitNode(declaratorNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var declarator *variableDeclarator
	if variableDeclarator, isVariableDeclarator := declaratorCommonNode.isVariableDeclarator(); isVariableDeclarator {
		declarator = variableDeclarator
	} else {
		return nil, fmt.Errorf("invalid ast: declarator is not a variable_declarator")
	}

	lexicalDeclaration.declarator = declarator
	lexicalDeclaration.kind = kind

	return &lexicalDeclaration, nil
}
