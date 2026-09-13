package parser

import (
	"path"
	"path/filepath"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TODO: needs despaghetti-ing

func ExtractComponentData(class *Class, node *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["decorator"] = func(node *sitter.Node, _ any, indexInParent uint, funcMap walk.VisitorFuncMap[any]) (any, error) {
		call := node.NamedChild(0)
		if call.Kind() != "call_expression" {
			return nil, nil
		}

		decoratorNameNode := call.ChildByFieldName("function")
		if decoratorNameNode == nil {
			return nil, nil
		}

		decoratorName := decoratorNameNode.Utf8Text(content)

		var err error

		switch dn := decoratorName; dn {
		case "Component":
			class.EnsureAngular()
			class.Snapshot().Angular.EnsureComponent()
			err = walkComponentDecoratorParams(class, node, content)
		case "Directive":
			class.EnsureAngular()
			class.Snapshot().Angular.EnsureDirective()
			err = walkDirectiveDecoratorParams(class, node, content)
		case "NgModule":
			class.EnsureAngular()
			class.Snapshot().Angular.EnsureModule()
			err = walkModuleDecoratorParams(class, node, content)
		case "Pipe":
			class.EnsureAngular()
			class.Snapshot().Angular.EnsurePipe()
			err = walkPipeDecoratorParams(class, node, content)
		}

		return nil, err
	}

	_, err := walk.WalkTypeScript(node, nil, funcMap)
	if err != nil {
		return err
	}

	dirDef := class.GetOwnDefinition(DIR_PROP)
	if dirDef != nil {
		handleCompiledDirectiveProp(class, dirDef)
	}

	modDef := class.GetOwnDefinition(MOD_PROP)
	if modDef != nil {
		handleCompiledModuleProp(class, modDef)
	}

	pipeDef := class.GetOwnDefinition(PIPE_PROP)
	if pipeDef != nil {
		handleCompiledPipeProp(class, pipeDef)
	}

	return nil
}

func extractProvider(node *sitter.Node, content []byte) *Provider {
	nodeType := node.Kind()
	if nodeType == "identifier" {
		provider := Provider{}

		reference := &Reference{Name: node.Utf8Text(content), Node: node}
		provider.Token = reference
		provider.Class = reference

		return &provider
	}

	if nodeType != "object" {
		return nil
	}

	provider := Provider{}

	for i := range node.NamedChildCount() {
		pair := node.NamedChild(i)
		key := pair.ChildByFieldName("key")
		value := pair.ChildByFieldName("value")

		if key == nil || value == nil {
			return nil
		}

		keyName := key.Utf8Text(content)
		valueText := value.Utf8Text(content)

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

func handleComponentKv(class *Class, vNode *sitter.Node, content []byte, keyName string) error {
	var err error

	switch kn := keyName; kn {
	case "imports":
		handleImportsComponentKv(class, vNode, content)
	case "providers":
		handleProvidersComponentKv(class, vNode, content)
	case "selector":
		handleSelectorComponentKv(class, vNode, content)
	case "templateUrl":
		err = handleTemplateUrlKv(class, vNode, content)
	}

	return err
}

func handleCompiledDirectiveProp(class *Class, def *Definition) {
	node := def.Node
	tNode := node.ChildByFieldName("type")
	if tNode == nil || tNode.Kind() != "type_annotation" {
		return
	}

	gTypeNode := tNode.NamedChild(0)
	if gTypeNode == nil || gTypeNode.Kind() != "generic_type" {
		return
	}

	args := gTypeNode.ChildByFieldName("type_arguments")
	if args == nil || args.Kind() != "type_arguments" {
		return
	}

	selectorsNode := args.NamedChild(1)
	if selectorsNode == nil || selectorsNode.Kind() != "literal_type" {
		return
	}

	inputMapNode := args.NamedChild(3)
	if inputMapNode == nil || inputMapNode.Kind() != "object_type" {
		return
	}

	outputMapNode := args.NamedChild(4)
	if outputMapNode == nil || outputMapNode.Kind() != "object_type" {
		return
	}

	class.EnsureAngular()
	class.Snapshot().Angular.EnsureDirective()

	selectorsStringNode := selectorsNode.NamedChild(0)
	if selectorsStringNode == nil || selectorsStringNode.Kind() != "string" {
		return
	}

	selectorsFragNode := selectorsStringNode.NamedChild(0)
	if selectorsFragNode == nil || selectorsFragNode.Kind() != "string_fragment" {
		return
	}

	selectorSplit := strings.SplitSeq(selectorsFragNode.Utf8Text([]byte(class.Snapshot().Content)), ",")
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

func handleCompiledModuleProp(class *Class, def *Definition) {
	node := def.Node
	tNode := node.ChildByFieldName("type")
	if tNode == nil || tNode.Kind() != "type_annotation" {
		return
	}

	gTypeNode := tNode.NamedChild(0)
	if gTypeNode == nil || gTypeNode.Kind() != "generic_type" {
		return
	}

	args := gTypeNode.ChildByFieldName("type_arguments")
	if args == nil || args.Kind() != "type_arguments" {
		return
	}

	declarationsNode := args.NamedChild(1)
	if declarationsNode == nil && declarationsNode.Kind() != "tuple_type" && declarationsNode.Kind() != "predefined_type" {
		return
	}

	importsNode := args.NamedChild(2)
	if importsNode == nil && importsNode.Kind() != "tuple_type" && importsNode.Kind() != "predefined_type" {
		return
	}

	exportsNode := args.NamedChild(3)
	if exportsNode == nil && exportsNode.Kind() != "tuple_type" && exportsNode.Kind() != "predefined_type" {
		return
	}

	class.EnsureAngular()
	class.Snapshot().Angular.EnsureModule()

	declarations := handleCompiledModuleArray(class, declarationsNode)
	imports := handleCompiledModuleArray(class, importsNode)
	exports := handleCompiledModuleArray(class, exportsNode)

	class.Update(func(data *classState) {
		data.Angular.Module.Declarations = &Value{ArrayValues: declarations, Type: "array"}
		data.Angular.Module.Imports = &Value{ArrayValues: imports, Type: "array"}
		data.Angular.Module.Exports = &Value{ArrayValues: exports, Type: "array"}
	})
}

func handleCompiledPipeProp(class *Class, def *Definition) {
	node := def.Node
	tNode := node.ChildByFieldName("type")
	if tNode == nil || tNode.Kind() != "type_annotation" {
		return
	}

	gTypeNode := tNode.NamedChild(0)
	if gTypeNode == nil || gTypeNode.Kind() != "generic_type" {
		return
	}

	args := gTypeNode.ChildByFieldName("type_arguments")
	if args == nil || args.Kind() != "type_arguments" {
		return
	}

	nameLiteralNode := args.NamedChild(1)
	if nameLiteralNode == nil || nameLiteralNode.Kind() != "literal_type" {
		return
	}

	nameNode := nameLiteralNode.NamedChild(0)
	if nameNode == nil || nameNode.Kind() != "string" {
		return
	}

	nameFragNode := nameNode.NamedChild(0)
	if nameFragNode == nil {
		return
	}

	class.EnsureAngular()
	class.Snapshot().Angular.EnsurePipe()

	class.Update(func(data *classState) {
		data.Angular.Pipe.Name = nameFragNode.Utf8Text([]byte(data.Content))
	})
}

func handleCompiledModuleArray(class *Class, arrayNode *sitter.Node) []*Value {
	results := []*Value{}

	if arrayNode.Kind() == "predefined_type" {
		return results
	}

	classSnapshot := class.Snapshot()
	file := classSnapshot.File
	content := []byte(classSnapshot.Content)

	for i := range arrayNode.NamedChildCount() {
		element := arrayNode.NamedChild(i)
		if element == nil || element.Kind() != "type_query" {
			continue
		}

		memberExpression := element.NamedChild(0)
		if memberExpression == nil || memberExpression.Kind() != "member_expression" {
			continue
		}

		property := memberExpression.ChildByFieldName("property")
		if property == nil || property.Kind() != "property_identifier" {
			continue
		}

		value := NodeToValue(file, property, content)
		results = append(results, value)
	}

	return results
}

func handleCompiledInputs(class *Class, inputMapNode *sitter.Node) {
	for i := range inputMapNode.NamedChildCount() {
		child := inputMapNode.NamedChild(i)
		if child == nil || child.Kind() != "property_signature" {
			continue
		}

		nameNode := child.ChildByFieldName("name")
		n := nameNode.Utf8Text([]byte(class.Snapshot().Content))
		name := strings.TrimSuffix(strings.TrimPrefix(n, "\""), "\"")

		tNode := child.ChildByFieldName("type")
		if tNode == nil || tNode.Kind() != "type_annotation" {
			continue
		}

		actualType := tNode.NamedChild(0)
		if actualType == nil {
			continue
		}

		switch actualType.Kind() {
		case "literal_type":
			{
				str := actualType.NamedChild(0)
				if str == nil {
					continue
				}

				def := class.GetDefinition(name)
				if def == nil {
					break
				}

				class.Update(func(data *classState) {
					s := str.Utf8Text([]byte(data.Content))
					def.Decorators = append(def.Decorators, Decorator{Arguments: []string{s}, IsAngular: true, Name: "Input"})
					data.Definitions.Set(def.Name, *def.Definition)
				})
			}
		case "object_type":
			{
				dec := Decorator{Arguments: []string{}, IsAngular: true, Name: "Input"}

				for k := range actualType.NamedChildCount() {
					propSigKey := actualType.NamedChild(k)
					if propSigKey == nil || propSigKey.Kind() != "property_signature" {
						continue
					}

					nnameNode := propSigKey.ChildByFieldName("name")
					if nnameNode == nil {
						continue
					}

					nnameFragNode := nnameNode.NamedChild(0)
					if nnameFragNode == nil {
						continue
					}

					vvalueNode := propSigKey.ChildByFieldName("type")
					if vvalueNode == nil {
						continue
					}

					vvalueFragNode := vvalueNode.NamedChild(0)
					if vvalueFragNode == nil {
						continue
					}

					content := []byte(class.Snapshot().Content)
					n := nnameFragNode.Utf8Text(content)
					switch n {
					case "alias":
						{
							v := vvalueFragNode.Utf8Text(content)
							dec.Arguments = append(dec.Arguments, v)
						}
					}
				}

				def := class.GetDefinition(name)
				if def == nil {
					break
				}

				class.Update(func(data *classState) {
					def.Decorators = append(def.Decorators, dec)
					data.Definitions.Set(def.Name, *def.Definition)
				})
			}
		}
	}
}

func handleCompiledOutputs(class *Class, inputMapNode *sitter.Node) {
	for i := range inputMapNode.NamedChildCount() {
		child := inputMapNode.NamedChild(i)
		if child.Kind() != "property_signature" {
			continue
		}

		nameNode := child.ChildByFieldName("name")
		n := nameNode.Utf8Text([]byte(class.Snapshot().Content))
		name := strings.TrimSuffix(strings.TrimPrefix(n, "\""), "\"")

		tNode := child.ChildByFieldName("type")
		if tNode == nil || tNode.Kind() != "type_annotation" {
			continue
		}

		actualType := tNode.NamedChild(0)
		if actualType == nil {
			continue
		}

		switch actualType.Kind() {
		case "literal_type":
			{
				str := actualType.NamedChild(0)
				if str == nil {
					continue
				}

				def := class.GetDefinition(name)
				if def == nil {
					break
				}

				class.Update(func(data *classState) {
					s := str.Utf8Text([]byte(data.Content))
					def.Decorators = append(def.Decorators, Decorator{Arguments: []string{s}, IsAngular: true, Name: "Output"})
				})
			}
		}
	}
}

func handleDirectiveKv(class *Class, vNode *sitter.Node, content []byte, keyName string) {
	switch kn := keyName; kn {
	case "imports":
		handleImportsDirectiveKv(class, vNode, content)
	case "providers":
		handleProvidersDirectiveKv(class, vNode, content)
	case "selector":
		handleSelectorDirectiveKv(class, vNode, content)
	}
}

func handleModuleKv(class *Class, vNode *sitter.Node, content []byte, keyName string) {
	switch kn := keyName; kn {
	case "imports":
		handleImportsModuleKv(class, vNode, content)
	case "exports":
		handleExportsKv(class, vNode, content)
	case "declarations":
		handleDeclarationsKv(class, vNode, content)
	case "providers":
		handleProvidersModuleKv(class, vNode, content)
	}
}

func handlePipeKv(class *Class, vNode *sitter.Node, content []byte, keyName string) {
	switch kn := keyName; kn {
	case "name":
		stringNode := vNode
		fragNode := stringNode.NamedChild(0)
		if fragNode == nil {
			return
		}

		class.Update(func(data *classState) {
			data.Angular.Pipe.Name = fragNode.Utf8Text(content)
		})
	}
}

func handleDeclarationsKv(class *Class, vNode *sitter.Node, content []byte) {
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Module.Declarations = NodeToValue(file, vNode, content)
	})
}

func handleExportsKv(class *Class, vNode *sitter.Node, content []byte) {
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Module.Exports = NodeToValue(file, vNode, content)
	})
}

func handleImportsComponentKv(class *Class, vNode *sitter.Node, content []byte) {
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Component.Imports = NodeToValue(file, vNode, content)
	})
}

func handleImportsDirectiveKv(class *Class, vNode *sitter.Node, content []byte) {
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Directive.Imports = NodeToValue(file, vNode, content)
	})
}

func handleImportsModuleKv(class *Class, vNode *sitter.Node, content []byte) {
	file := class.Snapshot().File
	class.Update(func(data *classState) {
		data.Angular.Module.Imports = NodeToValue(file, vNode, content)
	})
}

func handleProvidersComponentKv(class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Kind() != "array" {
		return
	}

	providers := make([]*Provider, 0)

	for i := range vNode.NamedChildCount() {
		provider := vNode.NamedChild(i)

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
	if vNode.Kind() != "array" {
		return
	}

	providers := make([]*Provider, 0)

	for i := range vNode.NamedChildCount() {
		provider := vNode.NamedChild(i)

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
	if vNode.Kind() != "array" {
		return
	}

	providers := make([]*Provider, 0)

	for i := range vNode.NamedChildCount() {
		provider := vNode.NamedChild(i)
		providers = append(providers, extractProvider(provider, content))
	}

	class.Update(func(data *classState) {
		data.Angular.Module.Providers = providers
	})
}

func handleSelectorComponentKv(class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Kind() != "string" {
		return
	}

	if vNode.NamedChildCount() != 1 {
		return
	}

	fragNode := vNode.NamedChild(0)
	if fragNode.Kind() != "string_fragment" {
		return
	}

	class.Update(func(data *classState) {
		selectors := fragNode.Utf8Text(content)

		split := strings.SplitSeq(selectors, ",")
		for s := range split {
			trimmed := strings.TrimSpace(s)

			data.Angular.Component.Selectors = append(data.Angular.Component.Selectors, trimmed)
		}

		data.Angular.Component.SelectorNode = fragNode
	})
}

func handleSelectorDirectiveKv(class *Class, vNode *sitter.Node, content []byte) {
	if vNode.Kind() != "string" {
		return
	}

	if vNode.NamedChildCount() != 1 {
		return
	}

	fragNode := vNode.NamedChild(0)
	if fragNode.Kind() != "string_fragment" {
		return
	}

	class.Update(func(data *classState) {
		selectors := fragNode.Utf8Text(content)

		split := strings.SplitSeq(selectors, ",")
		for s := range split {
			trimmed := strings.TrimSpace(s)

			data.Angular.Directive.Selectors = append(data.Angular.Directive.Selectors, trimmed)
		}

		data.Angular.Directive.SelectorNode = fragNode
	})
}

func handleTemplateUrlKv(class *Class, vNode *sitter.Node, content []byte) error {
	if vNode.Kind() != "string" {
		return nil
	}

	if vNode.NamedChildCount() != 1 {
		return nil
	}

	fragNode := vNode.NamedChild(0)
	if fragNode.Kind() != "string_fragment" {
		return nil
	}

	relativePath := fragNode.Utf8Text(content)
	if relativePath == "" {
		return nil
	}

	controllerDirectory := utils.PathDir(class.Snapshot().File.Filename())

	templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativePath))
	if err != nil {
		return err
	}

	if !utils.FileExists(templateFilePath) {
		return nil
	}

	class.Update(func(data *classState) {
		data.Angular.Component.TemplateUrl = templateFilePath
	})

	return nil
}

func walkComponentDecoratorParams(class *Class, node *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["pair"] = func(node *sitter.Node, _ any, indexInParent uint, funcMap walk.VisitorFuncMap[any]) (any, error) {
		keyNode := node.ChildByFieldName("key")
		if keyNode == nil {
			return nil, nil
		}

		keyName := keyNode.Utf8Text(content)
		valueNode := node.ChildByFieldName("value")
		if valueNode == nil {
			return nil, nil
		}

		err := handleComponentKv(class, valueNode, content, keyName)

		return nil, err
	}

	_, err := walk.WalkTypeScript(node, nil, funcMap)
	if err != nil {
		return err
	}

	return nil
}

func walkDirectiveDecoratorParams(class *Class, node *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["pair"] = func(node *sitter.Node, _ any, indexInParent uint, funcMap walk.VisitorFuncMap[any]) (any, error) {
		keyNode := node.ChildByFieldName("key")
		if keyNode == nil {
			return nil, nil
		}

		keyName := keyNode.Utf8Text(content)
		valueNode := node.ChildByFieldName("value")
		if valueNode == nil {
			return nil, nil
		}

		handleDirectiveKv(class, valueNode, content, keyName)

		return nil, nil
	}

	_, err := walk.WalkTypeScript(node, nil, funcMap)
	if err != nil {
		return err
	}

	return nil
}

func walkModuleDecoratorParams(class *Class, node *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["pair"] = func(node *sitter.Node, _ any, indexInParent uint, funcMap walk.VisitorFuncMap[any]) (any, error) {
		keyNode := node.ChildByFieldName("key")
		if keyNode == nil {
			return nil, nil
		}

		keyName := keyNode.Utf8Text(content)
		valueNode := node.ChildByFieldName("value")
		if valueNode == nil {
			return nil, nil
		}

		handleModuleKv(class, valueNode, content, keyName)

		return nil, nil
	}

	_, err := walk.WalkTypeScript(node, nil, funcMap)
	return err
}

func walkPipeDecoratorParams(class *Class, node *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["pair"] = func(node *sitter.Node, _ any, indexInParent uint, funcMap walk.VisitorFuncMap[any]) (any, error) {
		keyNode := node.ChildByFieldName("key")
		if keyNode == nil {
			return nil, nil
		}

		keyName := keyNode.Utf8Text(content)
		valueNode := node.ChildByFieldName("value")
		if valueNode == nil {
			return nil, nil
		}

		handlePipeKv(class, valueNode, content, keyName)

		return nil, nil
	}

	_, err := walk.WalkTypeScript(node, nil, funcMap)
	return err
}
