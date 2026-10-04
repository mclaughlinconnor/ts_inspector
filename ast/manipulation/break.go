package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type breakExpression struct {
	commonNode
	label nodeInterface
}

func (b *breakExpression) _buildCfgBlock() (bool, error) {
	cfg := b.getCfg()

	prevBlock := cfg.current
	afterBlock := cfg.peekBreakBlock()
	breakBlock := cfg.currentCfg().addBlock("Break block")

	if afterBlock == nil {
		return true, newAstError(b, "break stack is unexpectedly empty")
	}

	cfg.current = breakBlock

	cfg.addInstruction(instructionBranch, "", b, "")

	cfg.currentCfg().addEdge(prevBlock, breakBlock)
	cfg.currentCfg().addEdge(breakBlock, afterBlock)

	return true, nil
}

func (i *breakExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !i.isUnderCursor(offset) {
		return nil, false
	}

	if first && i.getKind() == kind {
		return i, true
	}

	if i.label != nil && i.label.isUnderCursor(offset) {
		return i.label.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || i.getKind() == kind {
		return i, true
	}

	return nil, false
}

func (e *breakExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(e); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if e.label != nil {
		if ret := e.label.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitBreak(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	breakExpression := breakExpression{commonNode: makeCommonNode("breakExpression", state, node)}
	breakExpression.setImpl(&breakExpression)

	var label nodeInterface
	var err error
	labelNode := node.ChildByFieldName("label")
	if labelNode != nil {
		label, err = walk.VisitNode(labelNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	breakExpression.label = label

	return &breakExpression, nil
}
