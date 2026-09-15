package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type identifier struct {
	commonNode
}

func visitIdentifier(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	identifier := identifier{commonNode: makeCommonNode("identifier", state, node)}

	return &identifier, nil
}
