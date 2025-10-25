package parser

import (
	"path"
	"path/filepath"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractComponentData(state *State, class *Class, node *sitter.Node, content []byte) {
	funcMap := walk.NewVisitorFuncsMap[any]()
	funcMap["decorator"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		call := node.NamedChild(0)
		if call.Type() != "call_expression" {
			return nil
		}

		decoratorNameNode := call.ChildByFieldName("function")
		if decoratorNameNode == nil {
			return nil
		}

		decoratorName := decoratorNameNode.Content(content)

		switch dn := decoratorName; dn {
		case "Component":
			class.EnsureAngular()
			class.Angular.EnsureComponent()
			walkComponentDecoratorParams(state, class, node, content)
		case "NgModule":
			class.EnsureAngular()
			class.Angular.EnsureModule()
			walkModuleDecoratorParams(state, class, node, content)
		}

		return nil
	}

	walk.Walk(node, nil, funcMap)
}

func walkComponentDecoratorParams(state *State, class *Class, node *sitter.Node, content []byte) {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["pair"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		keyNode := node.ChildByFieldName("key")
		if keyNode == nil {
			return nil
		}

		keyName := keyNode.Content(content)
		valueNode := node.ChildByFieldName("value")
		if valueNode == nil {
			return nil
		}

		handleComponentKv(state, class, valueNode, content, keyName)

		return nil
	}

	walk.Walk(node, nil, funcMap)
}

func walkModuleDecoratorParams(state *State, class *Class, node *sitter.Node, content []byte) {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["pair"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		keyNode := node.ChildByFieldName("key")
		if keyNode == nil {
			return nil
		}

		keyName := keyNode.Content(content)
		valueNode := node.ChildByFieldName("value")
		if valueNode == nil {
			return nil
		}

		handleModuleKv(state, class, valueNode, content, keyName)

		return nil
	}

	walk.Walk(node, nil, funcMap)
}

func handleComponentKv(state *State, class *Class, vNode *sitter.Node, content []byte, keyName string) {
	switch kn := keyName; kn {
	case "imports":
		handleImportsComponentKv(state, class, vNode, content)
	case "templateUrl":
		handleTemplateUrlKv(state, class, vNode, content)
	case "selector":
		handleSelectorKv(state, class, vNode, content)
	}
}

func handleModuleKv(state *State, class *Class, vNode *sitter.Node, content []byte, keyName string) {
	switch kn := keyName; kn {
	case "imports":
		handleImportsModuleKv(state, class, vNode, content)
	case "exports":
		handleExportsKv(state, class, vNode, content)
	case "declarations":
		handleDeclarationsKv(state, class, vNode, content)
	}
}

func handleDeclarationsKv(state *State, class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Type() != "array" {
		return
	}

	idents := make([]string, 0)

	for i := range vNode.NamedChildCount() {
		ident := vNode.NamedChild(int(i))
		if ident.Type() != "identifier" {
			continue
		}

		idents = append(idents, ident.Content(content))
	}

	class.Angular.Module.DeclarationsIdents = idents
}

func handleExportsKv(state *State, class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Type() != "array" {
		return
	}

	idents := make([]string, 0)

	for i := range vNode.NamedChildCount() {
		ident := vNode.NamedChild(int(i))
		if ident.Type() != "identifier" {
			continue
		}

		idents = append(idents, ident.Content(content))
	}

	class.Angular.Module.ExportsIdents = idents
}

func handleImportsComponentKv(state *State, class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Type() != "array" {
		return
	}

	idents := make([]string, 0)

	for i := range vNode.NamedChildCount() {
		ident := vNode.NamedChild(int(i))
		if ident.Type() != "identifier" {
			continue
		}

		idents = append(idents, ident.Content(content))
	}

	class.Angular.Component.ImportsIdents = idents
}

func handleImportsModuleKv(state *State, class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Type() != "array" {
		return
	}

	idents := make([]string, 0)

	for i := range vNode.NamedChildCount() {
		ident := vNode.NamedChild(int(i))
		if ident.Type() != "identifier" {
			continue
		}

		idents = append(idents, ident.Content(content))
	}

	class.Angular.Module.ImportsIdents = idents
}

func handleSelectorKv(state *State, class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Type() != "string" {
		return
	}

	if vNode.NamedChildCount() != 1 {
		return
	}

	fragNode := vNode.NamedChild(0)
	if fragNode.Type() != "string_fragment" {
		return
	}

	class.Angular.Component.Selector = fragNode.Content(content)
}

func handleTemplateUrlKv(state *State, class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Type() != "string" {
		return
	}

	if vNode.NamedChildCount() != 1 {
		return
	}

	fragNode := vNode.NamedChild(0)
	if fragNode.Type() != "string_fragment" {
		return
	}

	relativePath := fragNode.Content(content)
	if relativePath == "" {
		return
	}

	controllerDirectory := filepath.Dir(class.File.Filename())

	templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativePath))
	if err != nil {
		return
	}

	if !utils.FileExists(templateFilePath) {
		return
	}

	class.Angular.Component.TemplateUrl = templateFilePath
}
