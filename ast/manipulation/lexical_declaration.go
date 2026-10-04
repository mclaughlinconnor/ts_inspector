package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// nolint:unused
type lexicalDeclarationKind string

// nolint:unused
type lexicalDeclarationKindStruct struct {
	CONST lexicalDeclarationKind
	LET   lexicalDeclarationKind
}

// nolint:unused
var lexicalDeclarationKindEnum = lexicalDeclarationKindStruct{CONST: "const", LET: "let"}

type LexicalDeclaration struct {
	commonNode
	Declarator *VariableDeclarator
	IsExport   bool
	Kind       string
}

func (l *LexicalDeclaration) _buildCfgBlock() (bool, error) {
	l.getCfg().addInstruction(instructionAssign, l.Declarator.getNameText(), l, l.Declarator.getValueText())

	return true, nil
}

func (l *LexicalDeclaration) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !l.isUnderCursor(offset) {
		return nil, false
	}

	if first && l.getKind() == kind {
		return l, true
	}

	if l.Declarator != nil && l.Declarator.isUnderCursor(offset) {
		return l.Declarator.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || l.getKind() == kind {
		return l, true
	}

	return nil, false
}

func (l *LexicalDeclaration) visit(exec func(nodeInterface) int) int {
	if ret := exec(l); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := l.Declarator.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitlexicalDeclaration(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	lexicalDeclaration := LexicalDeclaration{commonNode: makeCommonNode("lexicalDeclaration", state, node)}
	lexicalDeclaration.setImpl(&lexicalDeclaration)

	kindNode := node.ChildByFieldName("kind")
	if kindNode == nil {
		return nil, newAstErrorTS(node, state, "missing kind")
	}

	declaratorNode := node.NamedChild(0)
	if declaratorNode == nil {
		return nil, newAstErrorTS(node, state, "missing declarator")
	}

	kind := kindNode.Utf8Text([]byte(lexicalDeclaration.getProgramText()))

	declaratorCommonNode, err := walk.VisitNode(declaratorNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var declarator *VariableDeclarator
	if variableDeclarator, isVariableDeclarator := IsNode[*VariableDeclarator](declaratorCommonNode); isVariableDeclarator {
		declarator = variableDeclarator
	} else {
		return nil, newAstError(variableDeclarator, "declarator is not a variable_declarator")
	}

	lexicalDeclaration.Declarator = declarator
	lexicalDeclaration.Kind = kind
	_, lexicalDeclaration.IsExport = IsNode[*ExportStatement](state)

	return &lexicalDeclaration, nil
}
