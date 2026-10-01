package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type methodSignature struct {
	commonField
	isAsync        bool
	isGenerator    bool
	isGetter       bool
	isSetter       bool
	parameters     *FormalParameters
	returnType     nodeInterface // type_annotation
	typeParameters *typeParameters
}

func (a *methodSignature) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	under, found := a.commonField.getAstNodeOfKindAtOffset(offset, kind, first)
	if found && under != a {
		return under, found
	}

	if a.parameters != nil && a.parameters.isUnderCursor(offset) {
		return a.parameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.typeParameters != nil && a.typeParameters.isUnderCursor(offset) {
		return a.typeParameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.returnType != nil && a.returnType.isUnderCursor(offset) {
		return a.returnType.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (a *methodSignature) visit(exec func(nodeInterface) int) int {
	if ret := a.commonField.visit(exec); ret == VisitAbort {
		return ret
	}

	if a.typeParameters != nil {
		if ret := a.typeParameters.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := a.parameters.visit(exec); ret == VisitAbort {
		return ret
	}

	if a.returnType != nil {
		if ret := a.returnType.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitMethodSignature(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	commonFieldCommonNode, err := visitCommonField(node, state, 0, funcMap)
	if err != nil {
		return nil, err
	}

	if commonFieldCommonNode == nil {
		return nil, fmt.Errorf("invalid ast: missing method signature")
	}

	commonField, isCommonField := IsNode[*commonField](commonFieldCommonNode)
	if !isCommonField {
		return nil, fmt.Errorf("invalid ast: method signature is't a method signature")
	}

	methodSignature := methodSignature{commonField: *commonField}
	methodSignature.kind = "methodSignature"
	methodSignature.setImpl(&methodSignature)

	nodes := map[string]nodeInterface{}

	for index, child := range node.Children(node.Walk()) {
		commonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}

		label := node.FieldNameForChild(uint32(index))
		if label == "" {
			label = child.Kind()
		}

		nodes[label] = commonNode
	}

	parametersCommonNode, found := nodes["parameters"]
	var parameters *FormalParameters
	if found {
		if formalParameters, isFormalParameters := IsNode[*FormalParameters](parametersCommonNode); isFormalParameters {
			parameters = formalParameters
		} else {
			return nil, fmt.Errorf("invalid ast: parameters isn't a formal parameters")
		}
	}

	typeParametersCommonNode, found := nodes["type_parameters"]
	var ttypeParameters *typeParameters
	if found {
		if typeParameters, isTypeParameters := IsNode[*typeParameters](typeParametersCommonNode); isTypeParameters {
			ttypeParameters = typeParameters
		} else {
			return nil, fmt.Errorf("invalid ast: typeParameters isn't a property identifier")
		}
	}

	methodSignature.isAsync = nodes["async"] != nil
	methodSignature.isGenerator = nodes["*"] != nil
	methodSignature.isGetter = nodes["get"] != nil
	methodSignature.isSetter = nodes["set"] != nil
	methodSignature.parameters = parameters
	methodSignature.returnType = nodes["return_type"]
	methodSignature.typeParameters = ttypeParameters

	return &methodSignature, nil
}
