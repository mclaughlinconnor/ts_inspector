package walk

import (
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func Walk[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T], lang *sitter.Language) T {
	optimizedMap := GenerateSymbolMap(lang, visitorFuncMap)

	return VisitNode(node, state, 0, optimizedMap)
}

func WalkPug[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) T {
	lang := utils.GetLanguage(utils.Pug)

	return Walk(node, state, visitorFuncMap, lang)
}

func WalkTypeScript[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) T {
	lang := utils.GetLanguage(utils.TypeScript)

	return Walk(node, state, visitorFuncMap, lang)
}

func VisitNode[T any](node *sitter.Node, state T, indexInParent int, visitorFuncMap VisitorFuncMap[T]) T {
	symbol := node.Symbol()

	function, found := visitorFuncMap[symbol]
	if !found {
		function = dummyVisitor
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

func FastVisitChildren[T any](node *sitter.Node, state T, funcMap VisitorFuncMap[T]) T {
	for i := range node.ChildCount() {
		index := int(i)
		state = VisitNode(node.Child(index), state, index, funcMap)
	}

	return state
}
