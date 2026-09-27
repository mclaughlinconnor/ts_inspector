package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type typeParameters struct {
	commonNode
	parameters []*typeParameter
}

func (t *typeParameters) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !t.isUnderCursor(offset) {
		return nil, false
	}

	if first && t.getKind() == kind {
		return t, true
	}

	for _, typeParameter := range t.parameters {
		if typeParameter != nil && typeParameter.isUnderCursor(offset) {
			return typeParameter.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	if kind == NULL_KIND || t.getKind() == kind {
		return t, true
	}

	return nil, false
}

func (t *typeParameters) visit(exec func(nodeInterface) int) int {
	if ret := exec(t); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, typeParameter := range t.parameters {
		if typeParameter == nil {
			continue
		}

		if ret := typeParameter.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitTypeParameters(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	typeParameters := typeParameters{commonNode: makeCommonNode("typeParameters", state, node)}
	typeParameters.setImpl(&typeParameters)

	parameters := []*typeParameter{}
	for _, child := range node.NamedChildren(node.Walk()) {
		childCommonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
		if childCommonNode == nil {
			return nil, err
		}

		if typeParameter, isTypeParameter := isNode[*typeParameter](childCommonNode); isTypeParameter {
			parameters = append(parameters, typeParameter)
		} else {
			return nil, fmt.Errorf("invalid ast: type parameter isn't a type parameter")
		}
	}

	typeParameters.parameters = parameters

	return &typeParameters, nil
}
