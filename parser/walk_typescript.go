package parser

import (
	"log"
	"ts_inspector/ast"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type typescriptWalkState struct {
	DefinitionStack  utils.Stack[Definition]
	InDecorator      bool
	TemplateFilename string
	*Class
}

type classWalkState struct {
	Classes   []*Class
	Decorator *sitter.Node
	Exports   []*Reference
	IsExport  bool
}

type funcWalkState struct {
	IsExport bool
}

type varWalkState struct {
	Kind     string // const/let/var
	IsExport bool
}

func IndexTypeScriptFileFromIndexer(state *State, filename string) error {
	var err error

	file, err := createFileIfNotExists(state, filename, "", 0)
	if err != nil {
		return err
	}
	file.ResetDeclarations()

	_, err = utils.ParseFile(false, file.Snapshot().Content, utils.TypeScript, nil, func(root *sitter.Node, fileContent []byte, _ any) (any, error) {
		err = extractFileImports(root, file)
		if err != nil {
			return nil, err
		}

		parseClasses(state, root, file)
		parseRootFunctions(state, root, file)
		parseRootVariables(state, root, file)

		return nil, nil
	})

	return err
}

func IndexTypeScriptFileFromLsp(state *State, uri string, languageId string, version int, content string, logger *log.Logger) error {
	var err error

	file, err := createFileIfNotExists(state, FilenameFromUri(uri), content, version)
	if err != nil {
		return err
	}
	file.ResetDeclarations()

	_, err = utils.ParseFile(false, file.Snapshot().Content, utils.TypeScript, nil, func(root *sitter.Node, fileContent []byte, _ any) (any, error) {
		err = extractFileImports(root, file) // todo need to reset imports too
		if err != nil {
			return nil, err
		}

		parseClasses(state, root, file)
		parseRootFunctions(state, root, file)
		parseRootVariables(state, root, file)

		return nil, nil
	})

	return err
}

func addUsage(class *Class, name string, node *sitter.Node, content []byte) {
	access := LocalAccess
	if isInConstructor(node, content) {
		access = ConstructorAccess
	}

	usageInstance := UsageInstance{access, class, node}

	class.SetUsageAccessType(name, usageInstance.Access)
	class.AppendUsage(name, &usageInstance)
	class.AppendDefinitionUsage(name, &usageInstance)
}

func extractClassName(root *sitter.Node, content []byte) (string, *sitter.Node) {
	type ret struct {
		text string
		node *sitter.Node
	}

	funcMap := walk.NewVisitorFuncsMap[ret]()

	classVisitor := func(node *sitter.Node, state ret, indexInParent int, funcMap walk.VisitorFuncMap[ret]) ret {
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return ret{}
		}

		return ret{text: nameNode.Content(content), node: nameNode}
	}

	funcMap["abstract_class_declaration"] = classVisitor
	funcMap["class_declaration"] = classVisitor
	funcMap["interface_declaration"] = classVisitor

	r := walk.WalkTypeScript(root, ret{}, funcMap)

	return r.text, r.node
}

func extractFileImports(root *sitter.Node, file *File) error {
	imports, err := ast.ExtractImports(root, []byte(file.Snapshot().Content))
	if err != nil {
		return err
	}

	file.Update(func(data *fileState) {
		data.Imports = imports
	})

	dynamicImports, err := ast.ExtractDynamicImports(root, []byte(file.Snapshot().Content))
	if err != nil {
		return err
	}

	file.Update(func(data *fileState) {
		data.DynamicImportPaths = dynamicImports
	})

	return nil
}

func extractMetadata(class *Class, root *sitter.Node, content []byte) {
	funcMap := walk.NewVisitorFuncsMap[*Class]()

	classVisitor := func(node *sitter.Node, state *Class, indexInParent int, funcMap walk.VisitorFuncMap[*Class]) *Class {
		for i := range node.NamedChildCount() {
			child := node.NamedChild(int(i))
			t := child.Type()

			if t == "type_parameters" {
				for ti := range child.NamedChildCount() {
					tp := child.NamedChild(int(ti))

					tpName := tp.ChildByFieldName("name")
					if tpName == nil {
						continue
					}

					state.Update(func(data *classState) {
						data.TypeParameters = append(data.TypeParameters, tpName.Content(content))
					})
				}
			}

			if t != "class_heritage" {
				continue
			}

			for i := range child.NamedChildCount() {
				clause := child.NamedChild(int(i))
				jt := clause.Type()

				if jt == "extends_clause" {
					extendsClause := clause
					identCount := int(extendsClause.NamedChildCount())
					extendsIdentifiers := make([]string, identCount)

					for i := range identCount {
						extendsIdentifiers[i] = extendsClause.NamedChild(i).Content(content)
					}

					state.Update(func(data *classState) {
						data.ExtendsIdentNames = extendsIdentifiers
					})
				} else if jt == "implements_clause" {
					implementsClause := clause
					identCount := int(implementsClause.NamedChildCount())
					implementsIdentifiers := make([]string, identCount)

					for i := range identCount {
						implementsIdentifiers[i] = implementsClause.NamedChild(i).Content(content)
					}

					state.Update(func(data *classState) {
						data.ImplementsIdentNames = implementsIdentifiers
					})
				}
			}

		}

		return nil
	}

	funcMap["abstract_class_declaration"] = classVisitor
	funcMap["class_declaration"] = classVisitor
	funcMap["interface_declaration"] = classVisitor

	walk.WalkTypeScript(root, class, funcMap)
}

func extractType(node *sitter.Node, content []byte) string {
	nodeTypes := []string{"type", "return_type"}

	for _, nodeType := range nodeTypes {
		typeNode := node.ChildByFieldName(nodeType)
		if typeNode != nil && typeNode.Type() == "type_annotation" {
			child := typeNode.NamedChild(0)
			if child != nil {
				return child.Content(content)
			}
		}
	}

	return ""
}

func extractTypeScriptDefinitions(class *Class, root *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[typescriptWalkState]()

	funcMap["method_definition"] = visitDefinition(content)
	funcMap["method_signature"] = visitDefinition(content)
	funcMap["property_definition"] = visitDefinition(content) // is this even a thing?
	funcMap["public_field_definition"] = visitDefinition(content)
	funcMap["required_parameter"] = visitDefinition(content)

	funcMap["decorator"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		state.DefinitionStack.Peek().Decorators = append(state.DefinitionStack.Peek().Decorators, handleDecorator(node, content))

		return state
	}
	funcMap["accessibility_modifier"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		a, err := CalculateAccessibilityFromString(node.Content(content))
		if err != nil {
			return state
		}

		state.DefinitionStack.Peek().AccessModifier = a

		return state
	}

	funcMap["static"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		state.DefinitionStack.Peek().Static = true

		return state
	}

	funcMap["override_modifier"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		state.DefinitionStack.Peek().Override = true

		return state
	}

	funcMap["readonly"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		state.DefinitionStack.Peek().Readonly = true

		return state
	}

	funcMap["async"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		state.DefinitionStack.Peek().Async = true

		return state
	}

	funcMap["generator"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		state.DefinitionStack.Peek().Generator = true

		return state

	}

	funcMap["set"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		state.DefinitionStack.Peek().Setter = true

		return state

	}

	funcMap["get"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if state.DefinitionStack.IsEmpty() {
			return state
		}

		state.DefinitionStack.Peek().Getter = true

		return state
	}

	s := typescriptWalkState{Class: class}
	s = walk.WalkTypeScript(root, s, funcMap)

	return nil
}

func extractTypeScriptUsages(class *Class, root *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[typescriptWalkState]()

	funcMap["member_expression"] = visitUsageExpression(content)
	funcMap["subscript_expression"] = visitUsageExpression(content)

	s := typescriptWalkState{Class: class}
	walk.WalkTypeScript(root, s, funcMap)

	return nil
}

func handleDecorator(node *sitter.Node, content []byte) Decorator {
	functionExpression := node.NamedChild(0)

	decoratorNameNode := functionExpression.ChildByFieldName("function")

	var decoratorName string
	var arguments []string

	if decoratorNameNode != nil { // @Decorator(...args)
		decoratorName = decoratorNameNode.Content(content)

		argumentsNode := functionExpression.ChildByFieldName("arguments")
		if argumentsNode != nil {
			for index := range argumentsNode.NamedChildCount() {
				argumentNode := argumentsNode.NamedChild(int(index))
				arguments = append(arguments, argumentNode.Content(content))
			}
		}
	} else { // @Decorator
		decoratorName = functionExpression.Content(content)
	}

	isAngularDecorator := IsAngularDecorator(decoratorName)

	return Decorator{arguments, isAngularDecorator, decoratorName}
}

func handleTemplate(state *State, class *Class, templateFilename string) error {
	class.Snapshot().Angular.EnsureComponent()
	class.Snapshot().Angular.Component.EnsureTemplate()

	return IndexPugFromTypeScript(state, class, templateFilename)
}

func isInConstructor(node *sitter.Node, content []byte) bool {
	current := node.Parent()

	for current != nil {
		if current.Type() == "method_definition" {
			if current.ChildByFieldName("name").Content(content) == "constructor" {
				return true
			}
		}

		current = current.Parent()
	}

	return false
}

func parseClasses(state *State, root *sitter.Node, file *File) {
	funcMap := walk.NewVisitorFuncsMap[classWalkState]()

	funcMap["export_statement"] = func(node *sitter.Node, classWalkState classWalkState, indexInParent int, funcMap walk.VisitorFuncMap[classWalkState]) classWalkState {
		classWalkState.IsExport = true

		decorator := node.ChildByFieldName("decorator")
		if decorator != nil {
			classWalkState.Decorator = decorator
		}

		classWalkState = walk.VisitNamedChildren(node, classWalkState, funcMap, false)
		classWalkState.IsExport = false
		classWalkState.Decorator = nil

		return classWalkState
	}

	classVisitor := func(node *sitter.Node, classWalkState classWalkState, indexInParent int, funcMap walk.VisitorFuncMap[classWalkState]) classWalkState {
		classContent := node.Content([]byte(file.Snapshot().Content))

		var class *Class
		_, err := utils.ParseFile(false, classContent, utils.TypeScript, nil, func(classRoot *sitter.Node, content []byte, _ any) (any, error) {
			uri := file.Snapshot().URI

			className, classNameNode := extractClassName(classRoot, content)
			if className == "" || classNameNode == nil {
				return nil, nil
			}

			var found bool
			class, found = state.GetClass(ClassId(uri, className))

			if !found {
				c := NewClass(classContent, file, node)
				c.Update(func(data *classState) {
					data.Name = className
					data.NameNode = classNameNode
				})

				class = &c
			} else {
				class.Reset()
				class.Update(func(data *classState) {
					data.Node = node
					data.Content = CStr2GoStr(content)
					data.Name = className
					data.NameNode = classNameNode
				})
			}

			extractMetadata(class, classRoot, []byte(class.Snapshot().Content))

			err := extractTypeScriptDefinitions(class, classRoot, []byte(class.Snapshot().Content))
			if err != nil {
				return class, err
			}

			err = extractTypeScriptUsages(class, classRoot, content)
			if err != nil {
				return class, err
			}

			if classWalkState.Decorator != nil {
				ExtractComponentData(class, classWalkState.Decorator, []byte(file.Snapshot().Content))

				for classWalkState.Decorator.NextSibling() != nil {
					if classWalkState.Decorator.NextSibling().Type() != "decorator" {
						break
					}

					classWalkState.Decorator = classWalkState.Decorator.NextSibling()

					ExtractComponentData(class, classWalkState.Decorator, []byte(file.Snapshot().Content))
				}
			} else {
				ExtractComponentData(class, node, []byte(file.Snapshot().Content))
			}

			if class.Snapshot().Angular != nil && class.Snapshot().Angular.Component != nil && class.Snapshot().Angular.Component.TemplateUrl != "" {
				err = handleTemplate(state, class, class.Snapshot().Angular.Component.TemplateUrl)
				if err != nil {
					return class, err
				}
			}

			return nil, nil
		})

		if err != nil || class == nil {
			return classWalkState
		}

		file.Update(func(data *fileState) {
			data.Classes = append(data.Classes, class)
			if classWalkState.IsExport {
				export := Reference{Node: node, Name: class.Snapshot().Name, Class: class}
				data.Exports = append(data.Exports, &export)
			}
		})

		return classWalkState
	}

	funcMap["abstract_class_declaration"] = classVisitor
	funcMap["class_declaration"] = classVisitor
	funcMap["interface_declaration"] = classVisitor

	classWalkState := classWalkState{}
	walk.WalkTypeScript(root, classWalkState, funcMap)
}

func parseRootFunctions(state *State, root *sitter.Node, file *File) {
	fileContent := []byte(file.Snapshot().Content)

	funcMap := walk.NewVisitorFuncsMap[funcWalkState]()

	funcMap["export_statement"] = func(node *sitter.Node, funcWalkState funcWalkState, indexInParent int, funcMap walk.VisitorFuncMap[funcWalkState]) funcWalkState {
		funcWalkState.IsExport = true
		walk.VisitNamedChildren(node, funcWalkState, funcMap, true)
		funcWalkState.IsExport = false

		return funcWalkState
	}

	funcVisitor := func(node *sitter.Node, funcWalkState funcWalkState, indexInParent int, _funcMap walk.VisitorFuncMap[funcWalkState]) funcWalkState {
		nameNode := node.ChildByFieldName("name")
		parametersNode := node.ChildByFieldName("parameters")
		bodyNode := node.ChildByFieldName("body")

		function := Function{Node: node, BodyNode: bodyNode, ParametersNode: parametersNode}

		if nameNode != nil {
			name := nameNode.Content(fileContent)
			function.Name = name
		}

		function.IsExport = funcWalkState.IsExport

		file.Update(func(data *fileState) {
			data.Functions = append(data.Functions, &function)
		})

		return funcWalkState
	}

	funcMap["function_declaration"] = funcVisitor
	funcMap["function_signature"] = funcVisitor

	funcMap["program"] = func(node *sitter.Node, funcWalkState funcWalkState, indexInParent int, funcMap walk.VisitorFuncMap[funcWalkState]) funcWalkState {
		walk.VisitNamedChildren(node, funcWalkState, funcMap, true)

		return funcWalkState
	}

	funcWalkState := funcWalkState{}
	walk.WalkTypeScriptShallow(root, funcWalkState, funcMap)
}

func parseRootVariables(state *State, root *sitter.Node, file *File) {
	fileContent := []byte(file.Snapshot().Content)

	funcMap := walk.NewVisitorFuncsMap[varWalkState]()

	funcMap["export_statement"] = func(node *sitter.Node, varWalkState varWalkState, indexInParent int, funcMap walk.VisitorFuncMap[varWalkState]) varWalkState {
		varWalkState.IsExport = true
		walk.VisitNamedChildren(node, varWalkState, funcMap, true)
		varWalkState.IsExport = false

		return varWalkState
	}

	declarationVisitor := func(node *sitter.Node, varWalkState varWalkState, indexInParent int, funcMap walk.VisitorFuncMap[varWalkState]) varWalkState {
		var kindNode *sitter.Node

		kindNode = node.ChildByFieldName("kind")
		if kindNode == nil {
			n := node.Child(0)
			if !n.IsNamed() {
				kindNode = n
			}
		}

		kind := kindNode.Content(fileContent)

		varWalkState.Kind = kind
		walk.VisitNamedChildren(node, varWalkState, funcMap, true)
		varWalkState.Kind = ""

		return varWalkState
	}

	funcMap["lexical_declaration"] = declarationVisitor
	funcMap["variable_declaration"] = declarationVisitor

	funcMap["variable_declarator"] = func(node *sitter.Node, varWalkState varWalkState, indexInParent int, _funcMap walk.VisitorFuncMap[varWalkState]) varWalkState {
		nameNode := node.ChildByFieldName("name")
		valueNode := node.ChildByFieldName("value")

		variable := Variable{Node: node}

		if nameNode != nil {
			name := nameNode.Content(fileContent)
			variable.Name = name
		}

		if valueNode != nil {
			variable.Value = NodeToValue(file, valueNode, fileContent)
		}

		variable.IsExport = varWalkState.IsExport
		variable.Kind = varWalkState.Kind

		file.Update(func(data *fileState) {
			if variable.IsExport {
				ref := Reference{Name: variable.Name, Node: node, Variable: &variable}
				data.Exports = append(data.Exports, &ref)
			}

			data.Variables = append(data.Variables, &variable)
		})

		return varWalkState
	}

	funcMap["program"] = func(node *sitter.Node, varWalkState varWalkState, indexInParent int, funcMap walk.VisitorFuncMap[varWalkState]) varWalkState {
		walk.VisitNamedChildren(node, varWalkState, funcMap, true)

		return varWalkState
	}

	varWalkState := varWalkState{}
	walk.WalkTypeScriptShallow(root, varWalkState, funcMap)
}

func visitDefinition(content []byte) walk.VisitorFunction[typescriptWalkState] {
	return func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		parentDefinition := state.DefinitionStack.Peek()
		var parentName string
		if parentDefinition != nil {
			parentName = parentDefinition.Name
		}

		// TODO: should use CreatePropertyDefinition
		definition := Definition{
			Decorators: []Decorator{}, Node: node, UsageAccess: NoAccess,
		}

		if node.Type() == "required_parameter" {
			definition.OriginFunctionName = parentName
		}

		state.DefinitionStack.Push(definition)

		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			state.DefinitionStack.Peek().Name = nameNode.Content(content)
		} else {
			nameNode := node.ChildByFieldName("pattern")
			if nameNode != nil {
				state.DefinitionStack.Peek().Name = nameNode.Content(content)
			}
		}

		if state.DefinitionStack.Peek().Name == "" {
			return state
		}

		state.DefinitionStack.Peek().Type = extractType(node, content)

		for i := range node.ChildCount() {
			index := int(i)
			state = walk.VisitNode(node.Child(index), state, index, funcMap, false)
		}

		finalDefinition := state.DefinitionStack.Pop()

		if node.Type() == "required_parameter" && finalDefinition.OriginFunctionName != "constuctor" && finalDefinition.IsLocalParam() {
			return state
		}

		state.Class.AddDefinition(*finalDefinition)

		return state
	}
}

func visitUsageExpression(content []byte) walk.VisitorFunction[typescriptWalkState] {
	return func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		objectNode := node.ChildByFieldName("object")

		// Only keep going if it's a this.abc or a Class.prototype.abc
		if objectNode.Type() != "this" {
			prototypeNode := objectNode.ChildByFieldName("property")
			if prototypeNode == nil || prototypeNode.Content(content) != "prototype" {
				for i := range node.NamedChildCount() {
					index := int(i)
					state = walk.VisitNode(node.NamedChild(index), state, index, funcMap, false)
				}

				return state
			}
		}

		varNode := node.ChildByFieldName("property")
		if varNode == nil {
			varNode = node.ChildByFieldName("index")
			varNode = varNode.NamedChild(0)

			if varNode == nil || varNode.Type() != "string_fragment" {
				for i := range node.NamedChildCount() {
					index := int(i)
					state = walk.VisitNode(node.NamedChild(index), state, index, funcMap, false)
				}

				return state
			}
		}

		varName := varNode.Content(content)
		addUsage(state.Class, varName, node, content)

		for i := range node.NamedChildCount() {
			index := int(i)
			state = walk.VisitNode(node.NamedChild(index), state, index, funcMap, false)
		}

		return state
	}
}
