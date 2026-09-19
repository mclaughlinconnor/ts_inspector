package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"
	"ts_inspector/interfaces"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type elseClause struct {
	commonNode
	statement nodeInterface
}

type ifStatement struct {
	commonNode
	alternative *elseClause
	condition   nodeInterface // While not all nodes are implemented, this cannot be invertable
	consequence nodeInterface
}

func (e *elseClause) isElseIf() bool {
	_, ok := e.statement.isIfStatement()

	return ok
}

func (e *elseClause) visit(exec func(nodeInterface) int) int {
	if ret := exec(e); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := e.statement.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func (i *ifStatement) flipElse() {
	if i.alternative == nil || i.alternative.isElseIf() {
		return
	}

	invertableCondition, ok := i.condition.isInvertable()
	if !ok {
		return
	}

	invertableCondition.invert()

	elseBody := i.alternative.statement.getText()
	consequenceBody := i.consequence.getText()

	i.alternative.statement.editText(consequenceBody)
	i.consequence.editText(elseBody)
}

func (i *ifStatement) getActions() []action {
	actions := []action{}

	if i.alternative != nil && !i.alternative.isElseIf() {
		actions = append(actions, action{Name: "Flip consequence and alternative branches", Perform: func() ([]utils.TextEdit, error) { return i.applyAction(i.flipElse) }})
	}

	return actions
}

func (i *ifStatement) getAnalysis() []interfaces.Analysis {
	analyses := []interfaces.Analysis{}

	if !i.alternative.isElseIf() {
		unaryExpression, isUnaryExpression := i.condition.isUnaryExpression()
		if isUnaryExpression && unaryExpression.Operator.getOperator() == unaryExpressionOperatorEnum.LNOT {
			analyses = append(analyses, interfaces.NewAnalysis("yoda speak", i.condition.getRange(), interfaces.AnalysisSeverity.Information, "This condition could be inverted for readability", nil))
		}
	}

	return analyses
}

func (i *ifStatement) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !i.isUnderCursor(offset) {
		return nil, false
	}

	if first && i.getKind() == kind {
		return i, true
	}

	if i.condition.isUnderCursor(offset) {
		return i.condition.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if i.consequence.isUnderCursor(offset) {
		return i.consequence.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if i.alternative != nil && i.alternative.isUnderCursor(offset) {
		return i.alternative.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || i.getKind() == kind {
		return i, true
	}

	return nil, false
}

func (i *ifStatement) visit(exec func(nodeInterface) int) int {
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

	if ret := i.consequence.visit(exec); ret == VisitAbort {
		return ret
	}

	if i.alternative != nil {
		if ret := i.alternative.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitElseClause(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	elseClause := elseClause{commonNode: makeCommonNode("elseClause", state, node)}
	elseClause.setImpl(&elseClause)

	statementNode := node.NamedChild(0)
	if statementNode == nil {
		return nil, fmt.Errorf("invalid ast: missing statement")
	}

	statement, err := walk.VisitNode(statementNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	elseClause.statement = statement

	return &elseClause, nil
}

func visitIfStatement(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	ifStatement := ifStatement{commonNode: makeCommonNode("ifStatement", state, node)}
	ifStatement.setImpl(&ifStatement)

	conditionNode := node.ChildByFieldName("condition")
	if conditionNode == nil {
		return nil, fmt.Errorf("invalid ast: missing condition")
	}

	consequenceNode := node.ChildByFieldName("consequence")
	if consequenceNode == nil {
		return nil, fmt.Errorf("invalid ast: missing consequence")
	}

	alternativeNode := node.ChildByFieldName("alternative")

	condition, err := walk.VisitNode(conditionNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	consequence, err := walk.VisitNode(consequenceNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var alternative *elseClause = nil
	if alternativeNode != nil {
		alternativeWalkState, err := walk.VisitNode(alternativeNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}

		alternativeElseClause, ok := alternativeWalkState.isElseClause()
		if !ok {
			return nil, fmt.Errorf("invalid ast: else branch is not an else node, %+v", alternativeWalkState)
		}

		alternative = alternativeElseClause
	}

	ifStatement.condition = condition
	ifStatement.consequence = consequence
	ifStatement.alternative = alternative

	return &ifStatement, nil
}
