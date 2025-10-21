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

func VisitNamedChildren[T any](node *sitter.Node, state T, funcMap VisitorFuncMap[T]) T {
	for i := range node.NamedChildCount() {
		index := int(i)
		state = VisitNode(node.NamedChild(index), state, index, funcMap)
	}

	return state
}

func VisitChildren[T any](node *sitter.Node, state T, funcMap VisitorFuncMap[T]) T {
	for i := range node.ChildCount() {
		index := int(i)
		state = VisitNode(node.Child(index), state, index, funcMap)
	}

	return state
}
