package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type callExpression struct {
	commonNode
	function  nodeInterface
	arguments nodeInterface // could be a `template_string`
}

func (i *callExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !i.isUnderCursor(offset) {
		return nil, false
	}

	if first && i.getKind() == kind {
		return i, true
	}

	if i.function.isUnderCursor(offset) {
		return i.function.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if i.arguments.isUnderCursor(offset) {
		return i.arguments.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || i.getKind() == kind {
		return i, true
	}

	return nil, false
}

func (e *callExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(e); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := e.function.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := e.arguments.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitCallExpression(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	callExpression := callExpression{commonNode: makeCommonNode("callExpression", state, node)}
	callExpression.setImpl(&callExpression)

	functionNode := node.ChildByFieldName("function")
	if functionNode == nil {
		return nil, fmt.Errorf("invalid ast: missing function identifier")
	}

	argumentsNode := node.ChildByFieldName("arguments")
	if argumentsNode == nil {
		return nil, fmt.Errorf("invalid ast: missing arguments")
	}

	function, err := walk.VisitNode(functionNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	arguments, err := walk.VisitNode(argumentsNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	callExpression.function = function
	callExpression.arguments = arguments

	return &callExpression, nil
}
