package walk

import sitter "github.com/smacker/go-tree-sitter"

var currentFuncMap any

func Walk[T any](node *sitter.Node, state T, visitorFuncMap VisitorFuncMap[T]) T {
	currentFuncMap = visitorFuncMap
	s := VisitNode(node, state, 0)
	return s
}

func VisitNode[T any](node *sitter.Node, state T, indexInParent int) T {
	funcMap := currentFuncMap.(VisitorFuncMap[T])
	t := node.Type()

	function, found := funcMap[t]
	if !found {
		function = funcMap[DefaultVisitorFuncKey]
	}

	state = function(node, state, indexInParent)

	return state
}
