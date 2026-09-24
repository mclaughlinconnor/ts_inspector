package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type propertyIdentifier struct {
	commonNode
}

func visitPropertyIdentifier(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	propertyIdentifier := propertyIdentifier{commonNode: makeCommonNode("propertyIdentifier", state, node)}
	propertyIdentifier.setImpl(&propertyIdentifier)

	return &propertyIdentifier, nil
}
