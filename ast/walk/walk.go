package walk

import (
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func Walk[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T], lang *sitter.Language, skipMissing bool) (T, error) {
	optimizedMap := GenerateSymbolMap(lang, visitorFuncMap)

	return VisitNode(node, state, 0, optimizedMap, skipMissing)
}

func WalkAngular[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) (T, error) {
	lang := utils.GetLanguage(utils.AngularContent)

	return Walk(node, state, visitorFuncMap, lang, false)
}

func WalkJavaScript[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) (T, error) {
	lang := utils.GetLanguage(utils.JavaScript)

	return Walk(node, state, visitorFuncMap, lang, false)
}

func WalkPug[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) (T, error) {
	lang := utils.GetLanguage(utils.Pug)

	return Walk(node, state, visitorFuncMap, lang, false)
}

func WalkTypeScript[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) (T, error) {
	lang := utils.GetLanguage(utils.TypeScript)

	return Walk(node, state, visitorFuncMap, lang, false)
}

func WalkTypeScriptShallow[T any](node *sitter.Node, state T, visitorFuncMap InitVisitorFuncMap[T]) (T, error) {
	lang := utils.GetLanguage(utils.TypeScript)

	return Walk(node, state, visitorFuncMap, lang, true)
}

func VisitNode[T any](node *sitter.Node, state T, indexInParent uint, visitorFuncMap VisitorFuncMap[T], skipMissing bool) (T, error) {
	kindId := node.KindId()

	function, found := visitorFuncMap[kindId]
	if !found {
		if skipMissing {
			return state, nil
		}

		function = dummyVisitor
	}

	return function(node, state, indexInParent, visitorFuncMap)
}

func VisitNamedChildren[T any](node *sitter.Node, state T, funcMap VisitorFuncMap[T], skipMissing bool) (T, error) {
	var err error

	for i := range node.NamedChildCount() {
		state, err = VisitNode(node.NamedChild(i), state, i, funcMap, skipMissing)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

func VisitChildren[T any](node *sitter.Node, state T, funcMap VisitorFuncMap[T], skipMissing bool) (T, error) {
	for i := range node.ChildCount() {
		state, err := VisitNode(node.Child(i), state, i, funcMap, skipMissing)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}
