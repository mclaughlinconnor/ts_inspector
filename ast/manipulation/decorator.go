package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type decorator struct {
	commonNode
}

func visitDecorator(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	decorator := decorator{commonNode: makeCommonNode("decorator", state, node)}
	decorator.setImpl(&decorator)

	return &decorator, nil
}
