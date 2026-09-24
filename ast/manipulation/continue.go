package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type continueExpression struct {
	commonNode
	label nodeInterface
}

func (c *continueExpression) _buildCfgBlock() (bool, error) {
	cfg := c.getCfg()

	prevBlock := cfg.current
	afterBlock := cfg.peekContinueBlock()
	breakBlock := cfg.currentCfg().addBlock("Continue block")

	if afterBlock == nil {
		return true, fmt.Errorf("continue stack is unexpectedly empty")
	}

	cfg.current = breakBlock

	cfg.addInstruction(instructionBranch, "", c, "")

	cfg.currentCfg().addEdge(prevBlock, breakBlock)
	cfg.currentCfg().addEdge(breakBlock, afterBlock)

	return true, nil
}

func (c *continueExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !c.isUnderCursor(offset) {
		return nil, false
	}

	if first && c.getKind() == kind {
		return c, true
	}

	if c.label.isUnderCursor(offset) {
		return c.label.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || c.getKind() == kind {
		return c, true
	}

	return nil, false
}

func (c *continueExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(c); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := c.label.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitContinue(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	continueExpression := continueExpression{commonNode: makeCommonNode("continueExpression", state, node)}
	continueExpression.setImpl(&continueExpression)

	labelNode := node.ChildByFieldName("label")
	if labelNode == nil {
		return nil, fmt.Errorf("invalid ast: missing label")
	}

	label, err := walk.VisitNode(labelNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	continueExpression.label = label

	return &continueExpression, nil
}
