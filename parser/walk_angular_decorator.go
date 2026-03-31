package parser

import (
	"path"
	"path/filepath"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

// TODO: needs despaghetti-ing

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
		case "Directive":
			class.EnsureAngular()
			class.Snapshot().Angular.EnsureDirective()
			walkDirectiveDecoratorParams(class, node, content)
		case "NgModule":
			class.EnsureAngular()
			class.Snapshot().Angular.EnsureModule()
			walkModuleDecoratorParams(class, node, content)
		}

		return nil
	}

	walk.WalkTypeScript(node, nil, funcMap)

	dirDef := class.GetDefinition(DIR_PROP)
	if dirDef != nil {
		handleCompiledDirectiveProp(class, dirDef)
	}
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
		handleSelectorComponentKv(class, vNode, content)
	case "templateUrl":
		handleTemplateUrlKv(class, vNode, content)
	}
}

func handleCompiledDirectiveProp(class *Class, def *Definition) {
	node := def.Node
	tNode := node.ChildByFieldName("type")
	if tNode == nil || tNode.Type() != "type_annotation" {
		return
	}

	gTypeNode := tNode.NamedChild(0)
	if gTypeNode == nil || gTypeNode.Type() != "generic_type" {
		return
	}

	args := gTypeNode.ChildByFieldName("type_arguments")
	if args == nil || args.Type() != "type_arguments" {
		return
	}

	selectorsNode := args.NamedChild(1)
	if selectorsNode == nil || selectorsNode.Type() != "literal_type" {
		return
	}

	inputMapNode := args.NamedChild(3)
	if inputMapNode == nil || inputMapNode.Type() != "object_type" {
		return
	}

	outputMapNode := args.NamedChild(4)
	if outputMapNode == nil || outputMapNode.Type() != "object_type" {
		return
	}

	class.EnsureAngular()
	class.Snapshot().Angular.EnsureDirective()

	selectorsStringNode := selectorsNode.NamedChild(0)
	if selectorsStringNode == nil || selectorsStringNode.Type() != "string" {
		return
	}

	selectorsFragNode := selectorsStringNode.NamedChild(0)
	if selectorsFragNode == nil || selectorsFragNode.Type() != "string_fragment" {
		return
	}

	selectorSplit := strings.SplitSeq(selectorsFragNode.Content([]byte(class.Snapshot().Content)), ",")
	for s := range selectorSplit {
		trimmed := strings.TrimSpace(s)

		class.Update(func(data *classState) {
			data.Angular.Directive.Selectors = append(data.Angular.Directive.Selectors, trimmed)
		})
	}

	class.Update(func(data *classState) {
		data.Angular.Directive.SelectorNode = selectorsNode
	})

	handleCompiledInputs(class, inputMapNode)
	handleCompiledOutputs(class, outputMapNode)
}

func handleCompiledInputs(class *Class, inputMapNode *sitter.Node) {
	for i := range inputMapNode.NamedChildCount() {
		child := inputMapNode.NamedChild(int(i))
		if child == nil || child.Type() != "property_signature" {
			continue
		}

		nameNode := child.ChildByFieldName("name")
		n := nameNode.Content([]byte(class.Snapshot().Content))
		name := strings.TrimSuffix(strings.TrimPrefix(n, "\""), "\"")

		tNode := child.ChildByFieldName("type")
		if tNode == nil || tNode.Type() != "type_annotation" {
			continue
		}

		actualType := tNode.NamedChild(0)
		if actualType == nil {
			continue
		}

		switch actualType.Type() {
		case "literal_type":
			{
				str := actualType.NamedChild(0)
				if str == nil {
					continue
				}

				def := class.GetDefinition(name)
				class.Update(func(data *classState) {
					def.Decorators = append(def.Decorators, Decorator{Arguments: []string{str.Content([]byte(data.Content))}, IsAngular: true, Name: "Input"})
				})
			}
		case "object_type":
			{
				dec := Decorator{Arguments: []string{}, IsAngular: true, Name: "Input"}

				for k := range actualType.NamedChildCount() {
					propSigKey := actualType.NamedChild(int(k))
					if propSigKey == nil || propSigKey.Type() != "property_signature" {
						continue
					}

					nnameNode := propSigKey.ChildByFieldName("name")
					if nnameNode == nil {
						continue
					}

					vvalueNode := propSigKey.ChildByFieldName("type")
					if vvalueNode == nil {
						continue
					}

					switch nnameNode.Content([]byte(class.Snapshot().Content)) {
					case "alias":
						{
							dec.Arguments = append(dec.Arguments, vvalueNode.Content([]byte(class.Snapshot().Content)))
						}
					}
				}

				def := class.GetDefinition(name)
				class.Update(func(data *classState) {
					def.Decorators = append(def.Decorators, dec)
				})
			}
		}
	}
}

func handleCompiledOutputs(class *Class, inputMapNode *sitter.Node) {
	for i := range inputMapNode.NamedChildCount() {
		child := inputMapNode.NamedChild(int(i))
		if child.Type() != "property_signature" {
			continue
		}

		nameNode := child.ChildByFieldName("name")
		name := nameNode.Content([]byte(class.Snapshot().Content))

		tNode := child.ChildByFieldName("type")
		if tNode == nil || tNode.Type() != "type_annotation" {
			continue
		}

		objectNode := tNode.NamedChild(0)
		if objectNode == nil || objectNode.Type() != "object_type" {
			continue
		}

		actualType := objectNode.NamedChild(0)
		if actualType == nil {
			continue
		}

		switch actualType.Type() {
		case "literal_type":
			{
				str := actualType.NamedChild(0)
				if str == nil {
					continue
				}

				def := class.GetDefinition(name)
				class.Update(func(data *classState) {
					def.Decorators = append(def.Decorators, Decorator{Arguments: []string{str.Content([]byte(data.Content))}, IsAngular: true, Name: "Input"})
				})
			}
		}
	}
}

func handleDirectiveKv(class *Class, vNode *sitter.Node, content []byte, keyName string) {
	switch kn := keyName; kn {
	case "imports":
		handleImportsDirectiveKv(class, vNode)
	case "providers":
		handleProvidersDirectiveKv(class, vNode, content)
	case "selector":
		handleSelectorDirectiveKv(class, vNode, content)
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
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Module.Exports = NodeToValue(file, vNode)
	})
}

func handleImportsComponentKv(class *Class, vNode *sitter.Node) {
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Component.Imports = NodeToValue(file, vNode)
	})
}

func handleImportsDirectiveKv(class *Class, vNode *sitter.Node) {
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Directive.Imports = NodeToValue(file, vNode)
	})
}

func handleImportsModuleKv(class *Class, vNode *sitter.Node) {
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

func handleProvidersDirectiveKv(class *Class, vNode *sitter.Node, content []byte) {
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
		data.Angular.Directive.Providers = providers
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

func handleSelectorComponentKv(class *Class, vNode *sitter.Node, content []byte) {
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

func handleSelectorDirectiveKv(class *Class, vNode *sitter.Node, content []byte) {
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

			data.Angular.Directive.Selectors = append(data.Angular.Directive.Selectors, trimmed)
		}

		data.Angular.Directive.SelectorNode = fragNode
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

func walkDirectiveDecoratorParams(class *Class, node *sitter.Node, content []byte) {
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

		handleDirectiveKv(class, valueNode, content, keyName)

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
