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

func (c *callExpression) _buildCfgBlock() (bool, error) {
	c.getCfg().addInstruction(instructionCall, "", c, "")

	if expressionStatement, isExpressionStatement := isNode[*expressionStatement](c.arguments); isExpressionStatement {
		_, err := expressionStatement.buildCfgBlock()
		if err != nil {
			return true, err
		}
	}

	return true, nil
}

func (c *callExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !c.isUnderCursor(offset) {
		return nil, false
	}

	if first && c.getKind() == kind {
		return c, true
	}

	if c.function.isUnderCursor(offset) {
		return c.function.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if c.arguments.isUnderCursor(offset) {
		return c.arguments.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || c.getKind() == kind {
		return c, true
	}

	return nil, false
}

func (c *callExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(c); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := c.function.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := c.arguments.visit(exec); ret == VisitAbort {
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
