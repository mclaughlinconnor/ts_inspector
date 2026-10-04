package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type methodDefinition struct {
	methodSignature
	body nodeInterface
}

func (m *methodDefinition) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !m.isUnderCursor(offset) {
		return nil, false
	}

	if first && m.getKind() == kind {
		return m, true
	}

	under, found := m.commonField.getAstNodeOfKindAtOffset(offset, kind, first)
	if found && under != m {
		return under, found
	}

	if m.body != nil && m.body.isUnderCursor(offset) {
		return m.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || m.getKind() == kind {
		return m, true
	}

	return nil, false
}

func (m *methodDefinition) _buildCfgBlock() (bool, error) {
	return true, buildFunctionCfg(m, m.name.getText(), m.body)
}

func (m *methodDefinition) visit(exec func(nodeInterface) int) int {
	if ret := m.methodSignature.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := m.body.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitMethodDefinition(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	methodSignatureCommonNode, err := visitMethodSignature(node, state, 0, funcMap)
	if err != nil {
		return nil, err
	}

	if methodSignatureCommonNode == nil {
		return nil, newAstError(methodSignatureCommonNode, "missing method signature")
	}

	methodSignature, isMethodSignature := IsNode[*methodSignature](methodSignatureCommonNode)
	if !isMethodSignature {
		return nil, newAstError(methodSignature, "method signature isn't a method signature: "+methodSignature.getKind())
	}

	methodDefinition := methodDefinition{methodSignature: *methodSignature}
	methodDefinition.kind = "methodDefinition"
	methodDefinition.setImpl(&methodDefinition)

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, newAstErrorTS(node, state, "missing body")
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	methodDefinition.body = body

	return &methodDefinition, nil
}
