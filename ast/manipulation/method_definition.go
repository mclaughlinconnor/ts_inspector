package manipulation

import (
	"fmt"
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

	if m.body != nil && m.body.isUnderCursor(offset) {
		return m.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if m.parameters != nil && m.parameters.isUnderCursor(offset) {
		return m.parameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if m.returnType != nil && m.returnType.isUnderCursor(offset) {
		return m.returnType.getAstNodeOfKindAtOffset(offset, kind, first)
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
		return nil, fmt.Errorf("invalid ast: missing method signature")
	}

	methodSignature, isMethodSignature := methodSignatureCommonNode.isMethodSignature()
	if !isMethodSignature {
		return nil, fmt.Errorf("invalid ast: method signature is't a method signature")
	}

	methodDefinition := methodDefinition{methodSignature: *methodSignature}
	methodDefinition.setImpl(&methodDefinition)
	methodDefinition.kind = "methodDefinition"

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, fmt.Errorf("invalid ast: missing body")
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	methodDefinition.body = body

	return &methodDefinition, nil
}
