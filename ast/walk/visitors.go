package walk

import sitter "github.com/tree-sitter/go-tree-sitter"

type VisitorFunction[T any] func(node *sitter.Node, state T, indexInParent uint, visitorFuncMap VisitorFuncMap[T]) (T, error)
type InitVisitorFuncMap[T any] map[string]VisitorFunction[T]
type VisitorFuncMap[T any] map[uint16]VisitorFunction[T]

func NewVisitorFuncsMap[T any]() InitVisitorFuncMap[T] {
	return map[string]VisitorFunction[T]{}
}

func GenerateSymbolMap[T any](lang *sitter.Language, stringMap map[string]VisitorFunction[T]) VisitorFuncMap[T] {
	optimizedMap := make(VisitorFuncMap[T])

	count := uint(lang.NodeKindCount())
	for i := range count {
		id := uint16(i)
		name := lang.NodeKindForId(id)

		handler, exists := stringMap[name]
		if exists {
			optimizedMap[id] = handler
		}
	}

	return optimizedMap
}

func dummyVisitor[T any](node *sitter.Node, state T, indexInParent uint, visitorFuncMap VisitorFuncMap[T]) (T, error) {
	return VisitNamedChildren(node, state, visitorFuncMap, false)
}
