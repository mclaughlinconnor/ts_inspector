package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type boolean struct {
	commonNode
}

func (b *boolean) getValue() bool {
	return b.getText() == "true"
}

func (b *boolean) hasConstantExpression() bool {
	return true
}

func (b *boolean) hasConstantFalse() bool {
	return !b.getValue()
}

func (b *boolean) hasConstantTrue() bool {
	return b.getValue()
}

func visitBoolean(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	boolean := boolean{commonNode: makeCommonNode("boolean", state, node)}
	boolean.setImpl(&boolean)

	return &boolean, nil
}
