package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type FunctionSignature struct {
	commonNode
	IsExport   bool
	Name       *Identifier
	Parameters *FormalParameters
	ReturnType nodeInterface // type_annotation
}

func (f *FunctionSignature) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !f.isUnderCursor(offset) {
		return nil, false
	}

	if first && f.getKind() == kind {
		return f, true
	}

	if f.Parameters != nil && f.Parameters.isUnderCursor(offset) {
		return f.Parameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if f.ReturnType != nil && f.ReturnType.isUnderCursor(offset) {
		return f.ReturnType.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || f.getKind() == kind {
		return f, true
	}

	return nil, false
}

func (f *FunctionSignature) visit(exec func(nodeInterface) int) int {
	if ret := exec(f); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := f.Parameters.visit(exec); ret == VisitAbort {
		return ret
	}

	if f.ReturnType != nil {
		if ret := f.ReturnType.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitFunctionSignature(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	functionSignature := FunctionSignature{commonNode: makeCommonNode("functionSignature", state, node)}
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

	var name *Identifier
	if identifier, isIdentifier := IsNode[*Identifier](nameCommonNode); isIdentifier {
		name = identifier
	} else {
		return nil, fmt.Errorf("invalid ast: name isn't an identifier")
	}

	parametersCommonNode, err := walk.VisitNode(parametersNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var parameters *FormalParameters
	if parametersCommonNode != nil {
		if formalParameters, isFormalParameters := IsNode[*FormalParameters](parametersCommonNode); isFormalParameters {
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

	_, functionSignature.IsExport = IsNode[*ExportStatement](state)
	functionSignature.Name = name
	functionSignature.Parameters = parameters
	functionSignature.ReturnType = returnType

	return &functionSignature, nil
}
