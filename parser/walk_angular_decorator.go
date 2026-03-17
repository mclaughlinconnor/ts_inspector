package parser

import (
	"path"
	"path/filepath"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractComponentData(class *Class, node *sitter.Node, content []byte) {
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
			class.Snapshot().Angular.EnsureComponent()
			walkComponentDecoratorParams(class, node, content)
		case "NgModule":
			class.EnsureAngular()
			class.Snapshot().Angular.EnsureModule()
			walkModuleDecoratorParams(class, node, content)
		}

		return nil
	}

	walk.WalkTypeScript(node, nil, funcMap)
}

func extractProvider(node *sitter.Node, content []byte) *Provider {
	nodeType := node.Type()
	if nodeType == "identifier" {
		provider := Provider{}

		reference := &Reference{Name: node.Content(content), Node: node}
		provider.Token = reference
		provider.Class = reference

		return &provider
	}

	if nodeType != "object" {
		return nil
	}

	provider := Provider{}

	for i := range node.NamedChildCount() {
		pair := node.NamedChild(int(i))
		key := pair.ChildByFieldName("key")
		value := pair.ChildByFieldName("value")

		if key == nil || value == nil {
			return nil
		}

		keyName := key.Content(content)
		valueText := value.Content(content)

		switch kn := keyName; kn {
		case "provide":
			provider.Token = &Reference{Name: valueText, Node: value}
		case "useClass":
			provider.Class = &Reference{Name: valueText, Node: value}
		case "useExisting":
			provider.Existing = &Reference{Name: valueText, Node: value}
		case "useFactory":
			provider.Factory = value
		case "useToken":
			provider.RefToken = &Reference{Name: valueText, Node: value}
		case "useValue":
			provider.Value = value
		}
	}

	return &provider
}

func handleComponentKv(class *Class, vNode *sitter.Node, content []byte, keyName string) {
	switch kn := keyName; kn {
	case "imports":
		handleImportsComponentKv(class, vNode)
	case "providers":
		handleProvidersComponentKv(class, vNode, content)
	case "selector":
		handleSelectorKv(class, vNode, content)
	case "templateUrl":
		handleTemplateUrlKv(class, vNode, content)
	}
}

func handleModuleKv(class *Class, vNode *sitter.Node, content []byte, keyName string) {
	switch kn := keyName; kn {
	case "imports":
		handleImportsModuleKv(class, vNode)
	case "exports":
		handleExportsKv(class, vNode)
	case "declarations":
		handleDeclarationsKv(class, vNode)
	case "providers":
		handleProvidersModuleKv(class, vNode, content)
	}
}

func handleDeclarationsKv(class *Class, vNode *sitter.Node) {
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Module.Declarations = NodeToValue(file, vNode)
	})
}

func handleExportsKv(class *Class, vNode *sitter.Node) {
	if vNode.Type() != "array" {
		return
	}

	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Module.Exports = NodeToValue(file, vNode)
	})
}

func handleImportsComponentKv(class *Class, vNode *sitter.Node) {
	if vNode.Type() != "array" {
		return
	}

	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Component.Imports = NodeToValue(file, vNode)
	})
}

func handleImportsModuleKv(class *Class, vNode *sitter.Node) {
	if vNode.Type() != "array" {
		return
	}

	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Module.Imports = NodeToValue(file, vNode)
	})
}

func handleProvidersComponentKv(class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Type() != "array" {
		return
	}

	providers := make([]*Provider, 0)

	for i := range vNode.NamedChildCount() {
		provider := vNode.NamedChild(int(i))

		p := extractProvider(provider, content)
		if p == nil {
			continue
		}

		providers = append(providers, p)
	}

	class.Update(func(data *classState) {
		data.Angular.Component.Providers = providers
	})
}

func handleProvidersModuleKv(class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Type() != "array" {
		return
	}

	providers := make([]*Provider, 0)

	for i := range vNode.NamedChildCount() {
		provider := vNode.NamedChild(int(i))
		providers = append(providers, extractProvider(provider, content))
	}

	class.Update(func(data *classState) {
		data.Angular.Module.Providers = providers
	})
}

func handleSelectorKv(class *Class, vNode *sitter.Node, content []byte) {
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

	class.Update(func(data *classState) {
		selectors := fragNode.Content(content)

		split := strings.SplitSeq(selectors, ",")
		for s := range split {
			trimmed := strings.TrimSpace(s)

			data.Angular.Component.Selectors = append(data.Angular.Component.Selectors, trimmed)
		}

		data.Angular.Component.SelectorNode = fragNode
	})
}

func handleTemplateUrlKv(class *Class, vNode *sitter.Node, content []byte) {
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

	controllerDirectory := filepath.Dir(class.Snapshot().File.Filename())

	templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativePath))
	if err != nil {
		return
	}

	if !utils.FileExists(templateFilePath) {
		return
	}

	class.Update(func(data *classState) {
		data.Angular.Component.TemplateUrl = templateFilePath
	})
}

func walkComponentDecoratorParams(class *Class, node *sitter.Node, content []byte) {
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

		handleComponentKv(class, valueNode, content, keyName)

		return nil
	}

	walk.WalkTypeScript(node, nil, funcMap)
}

func walkModuleDecoratorParams(class *Class, node *sitter.Node, content []byte) {
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

		handleModuleKv(class, valueNode, content, keyName)

		return nil
	}

	walk.WalkTypeScript(node, nil, funcMap)
}
