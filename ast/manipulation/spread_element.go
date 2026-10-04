package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type SpreadElement struct {
	commonNode
	expression nodeInterface
}

func (p *SpreadElement) getNode() impl[nodeInterface] {
	commonNode, _ := p.expression.(impl[nodeInterface])
	return commonNode
}

func (e *SpreadElement) visit(exec func(nodeInterface) int) int {
	if ret := exec(e); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := e.expression.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitSpreadElement(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	spreadElement := SpreadElement{commonNode: makeCommonNode("spreadElement", state, node)}
	spreadElement.setImpl(&spreadElement)

	if node.NamedChildCount() == 0 {
		return nil, newAstErrorTS(node, state, "no expression in spread element")
	}

	expressionNode := node.NamedChild(0)
	expression, err := walk.VisitNode(expressionNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	spreadElement.expression = expression

	return &spreadElement, nil
}
