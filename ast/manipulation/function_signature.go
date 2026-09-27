package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type functionSignature struct {
	commonNode
	name       *identifier
	parameters *formalParameters
	returnType nodeInterface // type_annotation
}

func (f *functionSignature) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !f.isUnderCursor(offset) {
		return nil, false
	}

	if first && f.getKind() == kind {
		return f, true
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

func (f *functionSignature) visit(exec func(nodeInterface) int) int {
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

	if f.returnType != nil {
		if ret := f.returnType.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitFunctionSignature(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	functionSignature := functionSignature{commonNode: makeCommonNode("functionSignature", state, node)}
	functionSignature.setImpl(&functionSignature)

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, fmt.Errorf("invalid ast: missing name")
	}

	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode == nil {
		return nil, fmt.Errorf("invalid ast: missing parameters")
	}

	returnTypeNode := node.ChildByFieldName("return_type")

	nameCommonNode, err := walk.VisitNode(nameNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var name *identifier
	if identifier, isIdentifier := IsNode[*identifier](nameCommonNode); isIdentifier {
		name = identifier
	} else {
		return nil, fmt.Errorf("invalid ast: name isn't an identifier")
	}

	parametersCommonNode, err := walk.VisitNode(parametersNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var parameters *formalParameters
	if parametersCommonNode != nil {
		if formalParameters, isFormalParameters := IsNode[*formalParameters](parametersCommonNode); isFormalParameters {
			parameters = formalParameters
		} else {
			return nil, fmt.Errorf("invalid ast: parameters isn't a formal parameters")
		}
	}

	var returnType nodeInterface
	if returnTypeNode != nil {
		returnType, err = walk.VisitNode(returnTypeNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	functionSignature.name = name
	functionSignature.parameters = parameters
	functionSignature.returnType = returnType

	return &functionSignature, nil
}
