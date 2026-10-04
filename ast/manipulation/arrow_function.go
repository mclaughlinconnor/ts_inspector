package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type arrowFunction struct {
	commonNode
	body       nodeInterface
	parameters *FormalParameters
	parameter  *Identifier
	returnType nodeInterface // type_annotation
}

func (a *arrowFunction) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	if a.body != nil && a.body.isUnderCursor(offset) {
		return a.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.parameter != nil && a.parameter.isUnderCursor(offset) {
		return a.parameter.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.parameters != nil && a.parameters.isUnderCursor(offset) {
		return a.parameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.returnType != nil && a.returnType.isUnderCursor(offset) {
		return a.returnType.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (f *arrowFunction) _buildCfgBlock() (bool, error) {
	return true, buildFunctionCfg(f, "anonymous", f.body)
}

func (a *arrowFunction) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if a.parameter != nil {
		if ret := a.parameter.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if a.parameters != nil {
		if ret := a.parameters.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := a.body.visit(exec); ret == VisitAbort {
		return ret
	}

	if a.returnType != nil {
		if ret := a.returnType.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitArrowFunction(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	arrowFunction := arrowFunction{commonNode: makeCommonNode("arrowFunction", state, node)}
	arrowFunction.setImpl(&arrowFunction)

	parameterNode := node.ChildByFieldName("parameter")
	parametersNode := node.ChildByFieldName("parameters")

	if parametersNode == nil && parameterNode == nil {
		return nil, newAstErrorTS(node, state, "missing parameter and parameters")
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, newAstErrorTS(node, state, "missing body")
	}

	returnTypeNode := node.ChildByFieldName("return_type")

	var parameter *Identifier
	if parameterNode != nil {
		parameterCommonNode, err := walk.VisitNode(parameterNode, state, 0, funcMap, false)
		if err != nil {
			return nil, newAstErrorETS(parameterNode, parameterCommonNode, err)
		}

		if identifier, isIdentifier := IsNode[*Identifier](parameterCommonNode); isIdentifier {
			parameter = identifier
		} else {
			return nil, newAstError(identifier, "parameter isn't an identifier: "+identifier.getKind())
		}
	}

	var parametersCommonNode nodeInterface
	if parametersNode != nil {
		parametersCommonNode, err := walk.VisitNode(parametersNode, state, 0, funcMap, false)
		if err != nil {
			return nil, newAstErrorETS(parametersNode, parametersCommonNode, err)
		}
	}

	var parameters *FormalParameters
	if parametersCommonNode != nil {
		if formalParameters, isFormalParameters := IsNode[*FormalParameters](parametersCommonNode); isFormalParameters {
			parameters = formalParameters
		} else {
			return nil, newAstError(formalParameters, "parameters isn't a formal parameters: "+formalParameters.getKind())
		}
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, newAstErrorETS(bodyNode, state, err)
	}

	var returnType nodeInterface
	if returnTypeNode != nil {
		returnType, err = walk.VisitNode(returnTypeNode, state, 0, funcMap, false)
		if err != nil {
			return nil, newAstErrorETS(returnTypeNode, state, err)
		}
	}

	arrowFunction.parameter = parameter
	arrowFunction.parameters = parameters
	arrowFunction.body = body
	arrowFunction.returnType = returnType

	return &arrowFunction, nil
}
