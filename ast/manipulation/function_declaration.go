package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type functionDeclaration struct {
	functionSignature
	body nodeInterface
}

func (f *functionDeclaration) _buildCfgBlock() (bool, error) {
	return true, buildFunctionCfg(f, f.name.getText(), f.body)
}

func (f *functionDeclaration) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !f.isUnderCursor(offset) {
		return nil, false
	}

	if first && f.getKind() == kind {
		return f, true
	}

	under, found := f.functionSignature.getAstNodeOfKindAtOffset(offset, kind, first)
	if found && under != f {
		return under, found
	}

	if f.body != nil && f.body.isUnderCursor(offset) {
		return f.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || f.getKind() == kind {
		return f, true
	}

	return nil, false
}

func (f *functionDeclaration) visit(exec func(nodeInterface) int) int {
	if ret := f.functionSignature.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := f.body.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitFunctionDeclaration(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	functionSignatureCommonNode, err := visitFunctionSignature(node, state, 0, funcMap)
	if err != nil {
		return nil, err
	}

	if functionSignatureCommonNode == nil {
		return nil, fmt.Errorf("invalid ast: missing function signature")
	}

	functionSignature, isFunctionSignature := IsNode[*functionSignature](functionSignatureCommonNode)
	if !isFunctionSignature {
		return nil, fmt.Errorf("invalid ast: function signature is't a function signature")
	}

	functionDeclaration := functionDeclaration{functionSignature: *functionSignature}
	functionDeclaration.kind = "functionDeclaration"
	functionDeclaration.setImpl(&functionDeclaration)

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, fmt.Errorf("invalid ast: missing body")
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	functionDeclaration.body = body

	return &functionDeclaration, nil
}
