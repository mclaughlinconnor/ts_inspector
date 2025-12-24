package walk

import (
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func Walk[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T], lang *sitter.Language, skipMissing bool) T {
	optimizedMap := GenerateSymbolMap(lang, visitorFuncMap)

	return VisitNode(node, state, 0, optimizedMap, skipMissing)
}

func WalkAngular[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) T {
	lang := utils.GetLanguage(utils.AngularContent)

	return Walk(node, state, visitorFuncMap, lang, false)
}

func WalkJavaScript[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) T {
	lang := utils.GetLanguage(utils.JavaScript)

	return Walk(node, state, visitorFuncMap, lang, false)
}

func WalkPug[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) T {
	lang := utils.GetLanguage(utils.Pug)

	return Walk(node, state, visitorFuncMap, lang, false)
}

func WalkTypeScript[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) T {
	lang := utils.GetLanguage(utils.TypeScript)

	return Walk(node, state, visitorFuncMap, lang, false)
}

func WalkTypeScriptShallow[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) T {
	lang := utils.GetLanguage(utils.TypeScript)

	return Walk(node, state, visitorFuncMap, lang, true)
}

func VisitNode[T any](node *sitter.Node, state T, indexInParent int, visitorFuncMap VisitorFuncMap[T], skipMissing bool) T {
	symbol := node.Symbol()

	function, found := visitorFuncMap[symbol]
	if !found {
		if skipMissing {
			return state
		}

		function = dummyVisitor
	}

	state = function(node, state, indexInParent, visitorFuncMap)

	return state
}

func VisitNamedChildren[T any](node *sitter.Node, state T, funcMap VisitorFuncMap[T], skipMissing bool) T {
	for i := range node.NamedChildCount() {
		index := int(i)
		state = VisitNode(node.NamedChild(index), state, index, funcMap, skipMissing)
	}

	return state
}

func VisitChildren[T any](node *sitter.Node, state T, funcMap VisitorFuncMap[T], skipMissing bool) T {
	for i := range node.ChildCount() {
		index := int(i)
		state = VisitNode(node.Child(index), state, index, funcMap, skipMissing)
	}

	return state
}
