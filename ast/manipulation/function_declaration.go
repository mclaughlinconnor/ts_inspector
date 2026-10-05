package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type FunctionDeclaration struct {
	FunctionSignature
	Body nodeInterface
}

func (f *FunctionDeclaration) _buildCfgBlock() (bool, error) {
	return true, buildFunctionCfg(f, f.Name.getText(), f.Body)
}

func (f *FunctionDeclaration) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !f.isUnderCursor(offset) {
		return nil, false
	}

	if first && f.getKind() == kind {
		return f, true
	}

	under, found := f.FunctionSignature.getAstNodeOfKindAtOffset(offset, kind, first)
	if found && under != f {
		return under, found
	}

	if f.Body != nil && f.Body.isUnderCursor(offset) {
		return f.Body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || f.getKind() == kind {
		return f, true
	}

	return nil, false
}

func (f *FunctionDeclaration) visit(exec func(nodeInterface) int) int {
	if ret := f.FunctionSignature.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := f.Body.visit(exec); ret == VisitAbort {
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
		return nil, newAstError(functionSignatureCommonNode, "missing function signature")
	}

	functionSignature, isFunctionSignature := IsNode[*FunctionSignature](functionSignatureCommonNode)
	if !isFunctionSignature {
		return nil, newAstError(functionSignatureCommonNode, "function signature isn't a function signature: "+functionSignatureCommonNode.getKind())
	}

	functionDeclaration := FunctionDeclaration{FunctionSignature: *functionSignature}
	functionDeclaration.kind = "functionDeclaration"
	functionDeclaration.setImpl(&functionDeclaration)

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, newAstErrorTS(node, state, "missing body")
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	functionDeclaration.Body = body

	return &functionDeclaration, nil
}
