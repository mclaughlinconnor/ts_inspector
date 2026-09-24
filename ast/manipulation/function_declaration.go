package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type functionDeclaration struct {
	commonNode
	body       nodeInterface
	name       *identifier
	parameters nodeInterface // todo: make it the formal_parameters node
	returnType nodeInterface // type_annotation
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

	if f.body != nil && f.body.isUnderCursor(offset) {
		return f.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if f.parameters != nil && f.parameters.isUnderCursor(offset) {
		return f.parameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if f.returnType != nil && f.returnType.isUnderCursor(offset) {
		return f.returnType.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || f.getKind() == kind {
		return f, true
	}

	return nil, false
}

func (f *functionDeclaration) visit(exec func(nodeInterface) int) int {
	if ret := exec(f); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := f.parameters.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := f.body.visit(exec); ret == VisitAbort {
		return ret
	}

	if f.returnType != nil {
		if ret := f.returnType.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitFunctionDeclaration(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	functionDeclaration := functionDeclaration{commonNode: makeCommonNode("functionDeclaration", state, node)}
	functionDeclaration.setImpl(&functionDeclaration)

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, fmt.Errorf("invalid ast: missing name")
	}

	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode == nil {
		return nil, fmt.Errorf("invalid ast: missing parameters")
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, fmt.Errorf("invalid ast: missing body")
	}

	returnTypeNode := node.ChildByFieldName("return_type")

	nameCommonNode, err := walk.VisitNode(nameNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var name *identifier
	if identifier, isIdentifier := nameCommonNode.isIdentifier(); isIdentifier {
		name = identifier
	} else {
		return nil, fmt.Errorf("invalid ast: name isn't an identifier")
	}

	parameters, err := walk.VisitNode(parametersNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var returnType nodeInterface
	if returnTypeNode != nil {
		returnType, err = walk.VisitNode(returnTypeNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	functionDeclaration.name = name
	functionDeclaration.parameters = parameters
	functionDeclaration.body = body
	functionDeclaration.returnType = returnType

	return &functionDeclaration, nil
}
