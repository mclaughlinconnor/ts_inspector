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

func (b *boolean) invert() {
	if b.getValue() {
		b.editText("false")
	} else {
		b.editText("true")
	}
}

func visitBoolean(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	boolean := boolean{commonNode: makeCommonNode("boolean", state, node)}

	return &boolean, nil
}
