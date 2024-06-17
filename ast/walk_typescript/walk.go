package walktypescript

import sitter "github.com/smacker/go-tree-sitter"

var currentFuncMap any

func Walk[T any](node *sitter.Node, state T, visitorFuncMap VisitorFuncMap[T]) T {
	currentFuncMap = visitorFuncMap
	s := VisitNode(node, state, 0)
	return s
}

func VisitNode[T any](node *sitter.Node, state T, indexInParent int) T {
	t := node.Type()
	f := currentFuncMap.(VisitorFuncMap[T])[t]
	state = f(node, state, indexInParent)

	return state
}
