package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type String struct {
	commonNode
}

func visitString(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	str := String{commonNode: makeCommonNode("string", state, node)}
	str.setImpl(&str)

	return &str, nil
}
