package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type unaryExpressionOperatorType = string

type unaryExpressonOperatorStruct struct {
	BNOT   unaryExpressionOperatorType
	DELETE unaryExpressionOperatorType
	LNOT   unaryExpressionOperatorType
	MINUS  unaryExpressionOperatorType
	PLUS   unaryExpressionOperatorType
	TYPEOF unaryExpressionOperatorType
	VOID   unaryExpressionOperatorType
}

var unaryExpressionOperatorEnum = unaryExpressonOperatorStruct{BNOT: "~", DELETE: "delete", LNOT: "!", MINUS: "-", PLUS: "+", TYPEOF: "typeof", VOID: "void"}

type unaryExpression struct {
	commonNode
	Operator unaryExpressionOperator
	Argument nodeInterface
}

type unaryExpressionOperator struct {
	commonNode
}

func (b *unaryExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !b.isUnderCursor(offset) {
		return nil, false
	}

	if first && b.getKind() == kind {
		return b, true
	}

	if b.Operator.isUnderCursor(offset) {
		return b.Operator.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if b.Argument.isUnderCursor(offset) {
		return b.Argument.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || b.getKind() == kind {
		return b, true
	}

	return nil, false
}

func (b *unaryExpression) invert() {
	b.Operator.invert()
}

func (b *unaryExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(b); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := b.Operator.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := b.Argument.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func (b *unaryExpressionOperator) getOperator() unaryExpressionOperatorType {
	return b.getText()
}

func (b *unaryExpressionOperator) invert() {
	switch b.getOperator() {
	case unaryExpressionOperatorEnum.BNOT:
		b.editText("")
	case unaryExpressionOperatorEnum.LNOT:
		b.editText("")
	}
}

func visitUnaryExpression(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	unaryExpression := unaryExpression{commonNode: makeCommonNode("unaryExpression", state, node)}
	unaryExpression.setImpl(&unaryExpression)

	operatorNode := node.ChildByFieldName("operator")
	if operatorNode == nil {
		return nil, fmt.Errorf("invalid ast: missing operator")
	}

	argumentNode := node.ChildByFieldName("argument")
	if argumentNode == nil {
		return nil, fmt.Errorf("invalid ast: missing argument")
	}

	operator := unaryExpressionOperator{commonNode: makeCommonNode("operator", state, operatorNode)}
	operator.setImpl(&operator)

	argument, err := walk.VisitNode(argumentNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	unaryExpression.Operator = operator
	unaryExpression.Argument = argument

	return &unaryExpression, nil
}
