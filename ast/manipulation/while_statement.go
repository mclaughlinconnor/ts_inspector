package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type whileStatement struct {
	commonNode
	condition nodeInterface
	body      nodeInterface
}

func (i *whileStatement) _buildCfgBlock() (bool, error) {
	cfg := i.getCfg()

	condBlock := cfg.currentCfg().addBlock("While condition block")
	bodyBlock := cfg.currentCfg().addBlock("While body block")
	afterBlock := cfg.currentCfg().addBlock("While after block")

	cfg.pushLoopBlocks(condBlock, afterBlock)

	cfg.addInstruction(instructionBranch, "", i, "")

	cfg.currentCfg().addEdge(cfg.current, condBlock)
	condBlock.Node = i.condition

	cfg.current = condBlock
	cfg.addInstruction(instructionBranch, "", i, "")

	if !condBlock.hasConstantTrue() {
		cfg.currentCfg().addEdge(condBlock, afterBlock)
	}

	if !condBlock.hasConstantFalse() {
		cfg.currentCfg().addEdge(cfg.current, bodyBlock)
	}

	bodyBlock.Node = i.body
	cfg.current = bodyBlock
	_, err := bodyBlock.Node.buildCfgBlock()
	if err != nil {
		return true, err
	}

	if len(cfg.current.After) == 0 {
		cfg.currentCfg().addEdge(cfg.current, condBlock)
	}

	cfg.current = afterBlock

	cfg.popLoopBlocks()

	return true, nil
}

func (i *whileStatement) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !i.isUnderCursor(offset) {
		return nil, false
	}

	if first && i.getKind() == kind {
		return i, true
	}

	if i.condition.isUnderCursor(offset) {
		return i.condition.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if i.body.isUnderCursor(offset) {
		return i.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || i.getKind() == kind {
		return i, true
	}

	return nil, false
}

func (i *whileStatement) visit(exec func(nodeInterface) int) int {
	if ret := exec(i); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := i.condition.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := i.body.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitWhileStatement(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	whileStatement := whileStatement{commonNode: makeCommonNode("whileStatement", state, node)}
	whileStatement.setImpl(&whileStatement)

	conditionNode := node.ChildByFieldName("condition")
	if conditionNode == nil {
		return nil, fmt.Errorf("invalid ast: missing condition")
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, fmt.Errorf("invalid ast: missing body")
	}

	condition, err := walk.VisitNode(conditionNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	whileStatement.condition = condition
	whileStatement.body = body

	return &whileStatement, nil
}
