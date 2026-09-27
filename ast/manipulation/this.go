package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type this struct {
	commonNode
}

func visitThis(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	this := this{commonNode: makeCommonNode("this", state, node)}
	this.setImpl(&this)

	return &this, nil
}
