package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Identifier struct {
	commonNode
}

func (i *Identifier) invert() {
	i.editText("!" + i.getText())
}

func visitIdentifier(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	identifier := Identifier{commonNode: makeCommonNode("identifier", state, node)}
	identifier.setImpl(&identifier)

	return &identifier, nil
}
