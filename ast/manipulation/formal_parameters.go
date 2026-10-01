package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type FormalParameters struct {
	commonNode
	parameters []nodeInterface
}

func (t *FormalParameters) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !t.isUnderCursor(offset) {
		return nil, false
	}

	if first && t.getKind() == kind {
		return t, true
	}

	for _, parameter := range t.parameters {
		if parameter != nil && parameter.isUnderCursor(offset) {
			return parameter.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	if kind == NULL_KIND || t.getKind() == kind {
		return t, true
	}

	return nil, false
}

func (t *FormalParameters) visit(exec func(nodeInterface) int) int {
	if ret := exec(t); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, parameter := range t.parameters {
		if parameter == nil {
			continue
		}

		if ret := parameter.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitFormalParameters(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	formalParameters := FormalParameters{commonNode: makeCommonNode("formalParameters", state, node)}
	formalParameters.setImpl(&formalParameters)

	parameters := []nodeInterface{}
	for _, child := range node.NamedChildren(node.Walk()) {
		childCommonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
		if childCommonNode == nil {
			return nil, err
		}

		parameters = append(parameters, childCommonNode)
	}

	formalParameters.parameters = parameters

	return &formalParameters, nil
}
