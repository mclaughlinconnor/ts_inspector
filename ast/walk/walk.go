package walk

import sitter "github.com/smacker/go-tree-sitter"

func Walk[T any](node *sitter.Node, state T, visitorFuncMap VisitorFuncMap[T]) T {
	s := VisitNode(node, state, 0, visitorFuncMap)
	return s
}

func VisitNode[T any](node *sitter.Node, state T, indexInParent int, visitorFuncMap VisitorFuncMap[T]) T {
	t := node.Type()

	function, found := visitorFuncMap[t]
	if !found {
		function = visitorFuncMap[DefaultVisitorFuncKey]
	}

	state = function(node, state, indexInParent, visitorFuncMap)

	return state
}
