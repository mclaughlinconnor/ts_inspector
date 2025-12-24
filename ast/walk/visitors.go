package walk

import sitter "github.com/smacker/go-tree-sitter"

type VisitorFunction[T any] func(node *sitter.Node, state T, indexInParent int, visitorFuncMap VisitorFuncMap[T]) T
type InitVisitorFuncMap[T any] map[string]VisitorFunction[T]
type VisitorFuncMap[T any] map[sitter.Symbol]VisitorFunction[T]

func NewVisitorFuncsMap[T any]() InitVisitorFuncMap[T] {
	return map[string]VisitorFunction[T]{}
}

func GenerateSymbolMap[T any](lang *sitter.Language, stringMap map[string]VisitorFunction[T]) VisitorFuncMap[T] {
	optimizedMap := make(VisitorFuncMap[T])

	count := uint32(lang.SymbolCount())
	for i := range count {
		symbolID := sitter.Symbol(i)
		name := lang.SymbolName(symbolID)

		handler, exists := stringMap[name]
		if exists {
			optimizedMap[symbolID] = handler
		}
	}

	return optimizedMap
}

func dummyVisitor[T any](node *sitter.Node, state T, indexInParent int, visitorFuncMap VisitorFuncMap[T]) T {
	return VisitNamedChildren(node, state, visitorFuncMap, false)
}
