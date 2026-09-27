package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type ttype struct {
	commonNode
}

func visitType(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	ttype := ttype{commonNode: makeCommonNode("type", state, node)}
	ttype.setImpl(&ttype)

	return &ttype, nil
}
