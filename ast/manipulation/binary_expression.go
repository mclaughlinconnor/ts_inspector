package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"
	"ts_inspector/interfaces"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type binaryExpressionOperatorType = string

type binaryExpressonOperatorStruct struct {
	AND  binaryExpressionOperatorType
	EEQ  binaryExpressionOperatorType
	EQ   binaryExpressionOperatorType
	GT   binaryExpressionOperatorType
	GTE  binaryExpressionOperatorType
	LT   binaryExpressionOperatorType
	LTE  binaryExpressionOperatorType
	NEEQ binaryExpressionOperatorType
	NEQ  binaryExpressionOperatorType
	OR   binaryExpressionOperatorType
}

var binaryExpressionOperatorEnum = binaryExpressonOperatorStruct{AND: "&&", EEQ: "===", EQ: "==", GT: ">", GTE: ">=", LT: "<", LTE: "<=", NEEQ: "!==", NEQ: "!=", OR: "||"}

type binaryExpression struct {
	commonNode
	Left     nodeInterface
	Operator binaryExpressionOperator
	Right    nodeInterface
}

type binaryExpressionOperator struct {
	commonNode
}

func (b *binaryExpression) deMorgans() {
	b.invert()
	b.editText("!(" + b.getText() + ")")
}

func (b *binaryExpression) getActions() []action {
	return []action{
		{Name: "Apply deMorgan's", Perform: func() ([]utils.TextEdit, error) { return b.applyAction(b.deMorgans) }},
		{Name: "Remove redundant terms from binary expression", Perform: func() ([]utils.TextEdit, error) { return b.applyAction(func() { b.removeRedundant() }) }},
		{Name: "Invert condition", Perform: func() ([]utils.TextEdit, error) { return b.applyAction(b.invert) }},
	}
}

func (b *binaryExpression) getAnalysis() []interfaces.Analysis {
	analyses := []interfaces.Analysis{}

	leftBoolean, leftIsBoolean := isNode[*boolean](b.Left)
	rightBoolean, rightIsBoolean := isNode[*boolean](b.Right)

	if !leftIsBoolean && !rightIsBoolean {
		return analyses
	}

	if leftIsBoolean {
		var r utils.Range
		if leftBoolean.getValue() == true && b.Operator.getOperator() == binaryExpressionOperatorEnum.OR {
			r = b.Right.getRange()
		} else {
			r = b.Left.getRange()
		}

		analyses = append(analyses, interfaces.NewAnalysis("redundant", r, interfaces.AnalysisSeverity.Error, "This part of the binary expression is redundant", nil))
	}

	if rightIsBoolean {
		var r utils.Range
		if rightBoolean.getValue() == true && b.Operator.getOperator() == binaryExpressionOperatorEnum.OR {
			r = b.Left.getRange()
		} else {
			r = b.Right.getRange()
		}

		analyses = append(analyses, interfaces.NewAnalysis("redundant", r, interfaces.AnalysisSeverity.Error, "This part of the binary expression is redundant", nil))
	}

	return analyses
}

func (b *binaryExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !b.isUnderCursor(offset) {
		return nil, false
	}

	if first && b.getKind() == kind {
		return b, true
	}

	if b.Left.isUnderCursor(offset) {
		return b.Left.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if b.Operator.isUnderCursor(offset) {
		return b.Operator.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if b.Right.isUnderCursor(offset) {
		return b.Right.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || b.getKind() == kind {
		return b, true
	}

	return nil, false
}

func (b *binaryExpression) hasConstantExpression() bool {
	if b.Operator.getOperator() != binaryExpressionOperatorEnum.AND {
		return false
	}

	return b.Left.hasConstantExpression() && b.Right.hasConstantExpression()
}

func (b *binaryExpression) hasConstantFalse() bool {
	if b.Operator.getOperator() != binaryExpressionOperatorEnum.AND {
		return false
	}

	return b.Left.hasConstantFalse() && b.Right.hasConstantFalse()
}

func (b *binaryExpression) hasConstantTrue() bool {
	if b.Operator.getOperator() != binaryExpressionOperatorEnum.AND {
		return false
	}

	return b.Left.hasConstantTrue() && b.Right.hasConstantTrue()
}

func (b *binaryExpression) invert() {
	left, ok := isNode[*binaryExpression](b.Left)
	if ok {
		left.invert()
	}

	b.Operator.invert()

	right, ok := isNode[*binaryExpression](b.Right)
	if ok {
		right.invert()
	}
}

// Returns true when the entire node was made blank
func (b *binaryExpression) removeRedundant() bool {
	operatorNode := b.Operator
	operator := operatorNode.getOperator()

	if operator != binaryExpressionOperatorEnum.OR && operator != binaryExpressionOperatorEnum.AND {
		return false
	}

	binaryLeft, leftIsBinary := isNode[*binaryExpression](b.Left)
	binaryRight, rightIsBinary := isNode[*binaryExpression](b.Right)

	leftIsRedundant := false
	if leftIsBinary {
		leftIsRedundant = binaryLeft.removeRedundant()
	}

	rightIsRedundant := false
	if rightIsBinary {
		rightIsRedundant = binaryRight.removeRedundant()
	}

	if leftIsBinary && rightIsBinary {
		if leftIsRedundant && rightIsRedundant {
			b.editText("")
			return true
		}

		return false
	}

	if boolean, isRedundant := isNode[*boolean](b.Left); isRedundant {
		if boolean.getValue() == true && b.Operator.getOperator() == binaryExpressionOperatorEnum.OR {
			rightIsRedundant = isRedundant
		} else {
			leftIsRedundant = isRedundant
		}
	}

	if boolean, isRedundant := isNode[*boolean](b.Right); isRedundant {
		if boolean.getValue() == true && b.Operator.getOperator() == binaryExpressionOperatorEnum.OR {
			leftIsRedundant = isRedundant
		} else {
			rightIsRedundant = isRedundant
		}
	}

	if leftIsRedundant && rightIsRedundant {
		b.editText("")
		return true
	}

	if leftIsRedundant {
		b.editText(b.Right.getText())
		return false
	}

	if rightIsRedundant {
		b.editText(b.Left.getText())
		return false
	}

	return false
}

func (b *binaryExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(b); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := b.Left.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := b.Operator.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := b.Right.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func (b *binaryExpressionOperator) getOperator() binaryExpressionOperatorType {
	return b.getText()
}

func (b *binaryExpressionOperator) invert() {
	switch b.getOperator() {
	case binaryExpressionOperatorEnum.AND:
		b.editText("||")
	case binaryExpressionOperatorEnum.EEQ:
		b.editText("!==")
	case binaryExpressionOperatorEnum.EQ:
		b.editText("!=")
	case binaryExpressionOperatorEnum.GT:
		b.editText("<=")
	case binaryExpressionOperatorEnum.GTE:
		b.editText("<")
	case binaryExpressionOperatorEnum.LT:
		b.editText(">=")
	case binaryExpressionOperatorEnum.LTE:
		b.editText(">=")
	case binaryExpressionOperatorEnum.NEEQ:
		b.editText("===")
	case binaryExpressionOperatorEnum.NEQ:
		b.editText("==")
	case binaryExpressionOperatorEnum.OR:
		b.editText("&&")
	}
}

func visitBinaryExpression(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	binaryExpression := binaryExpression{commonNode: makeCommonNode("binaryExpression", state, node)}
	binaryExpression.setImpl(&binaryExpression)

	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return nil, fmt.Errorf("invalid ast: missing left")
	}

	operatorNode := node.ChildByFieldName("operator")
	if operatorNode == nil {
		return nil, fmt.Errorf("invalid ast: missing operator")
	}

	rightNode := node.ChildByFieldName("right")
	if rightNode == nil {
		return nil, fmt.Errorf("invalid ast: missing right")
	}

	left, err := walk.VisitNode(leftNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	operator := binaryExpressionOperator{commonNode: makeCommonNode("operator", state, operatorNode)}
	operator.setImpl(&operator)

	right, err := walk.VisitNode(rightNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	binaryExpression.Left = left
	binaryExpression.Operator = operator
	binaryExpression.Right = right

	return &binaryExpression, nil
}
