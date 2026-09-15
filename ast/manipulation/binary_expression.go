package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"
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
		{Name: "deMorgans", Perform: func() []utils.TextEdit { return b.applyAction(b.deMorgans) }},
	}
}

func (b *binaryExpression) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return b.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (b *binaryExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	node := b.getTsNode()
	if node.StartByte() > offset || offset >= node.EndByte() {
		return nil, false
	}

	if first && b.getKind() == kind {
		return b, true
	}

	if b.Left != nil {
		leftNode := b.Left.getTsNode()
		if leftNode.StartByte() <= offset && offset < leftNode.EndByte() {
			return b.Left.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	operatorNode := b.Operator.getTsNode()
	if operatorNode.StartByte() <= offset && offset < operatorNode.EndByte() {
		return b.Operator.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if b.Right != nil {
		rightNode := b.Right.getTsNode()
		if rightNode.StartByte() <= offset && offset < rightNode.EndByte() {
			return b.Right.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	if kind == NULL_KIND || b.getKind() == kind {
		return b, true
	}

	return nil, false
}

func (b *binaryExpression) invert() {
	left, ok := b.Left.(invertableNodeInterface)
	if ok {
		left.invert()
	}

	b.Operator.invert()

	right, ok := b.Right.(invertableNodeInterface)
	if ok {
		right.invert()
	}
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

func (c *binaryExpressionOperator) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return c.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (c *binaryExpressionOperator) getAstNodeOfKindAtOffset(offset uint, kind string, _ bool) (nodeInterface, bool) {
	if kind != NULL_KIND && c.getKind() != kind {
		return nil, false
	}

	node := c.getTsNode()
	if node.StartByte() <= offset && offset < node.EndByte() {
		return c, true
	}

	return nil, false
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

func (b *binaryExpressionOperator) visit(exec func(nodeInterface) int) int {
	if ret := exec(b); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return VisitContinue
}


func visitBinaryExpression(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	binaryExpression := binaryExpression{commonNode: makeCommonNode("binaryExpression", state, node)}

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

	right, err := walk.VisitNode(rightNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	if left.getId() != state.getId() {
		binaryExpression.Left = left
	}

	binaryExpression.Operator = operator

	if right.getId() != state.getId() {
		binaryExpression.Right = right
	}

	return &binaryExpression, nil
}

func visitBoolean(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	boolean := boolean{commonNode: makeCommonNode("boolean", state, node)}

	return &boolean, nil
}
