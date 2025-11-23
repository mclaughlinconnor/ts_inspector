package walk

import sitter "github.com/smacker/go-tree-sitter"

type VisitorFunction[T any] func(node *sitter.Node, state T, indexInParent int, visitorFuncMap VisitorFuncMap[T]) T
type VisitorFuncMap[T any] map[string]VisitorFunction[T]

var DefaultVisitorFuncKey = "__ts_inspector_default"

func NewVisitorFuncsMap[T any]() VisitorFuncMap[T] {
	var visitorFuncs = VisitorFuncMap[T]{
		"__ts_inspector_default": dummyVisitor[T],
	}

	dst := make(map[string]VisitorFunction[T], len(visitorFuncs))

	for k, v := range visitorFuncs {
		dst[k] = v
	}

	return dst
}

func dummyVisitor[T any](node *sitter.Node, state T, indexInParent int, visitorFuncMap VisitorFuncMap[T]) T {
	return VisitNamedChildren(node, state, visitorFuncMap)
}
