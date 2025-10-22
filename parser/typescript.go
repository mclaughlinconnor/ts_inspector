package parser

import (
	"log"
	"path"
	"path/filepath"
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

func extractFileImports(root *sitter.Node, file *File) error {
	imports, err := ast.ExtractImports(root, []byte(file.Content))
	if err != nil {
		return err
	}

	file.Imports = append(file.Imports, imports...)

	return nil
}

func parseClasses(state *State, root *sitter.Node, file *File) {
	funcMap := walk.NewVisitorFuncsMap[classWalkState]()

	funcMap["export_statement"] = func(node *sitter.Node, classWalkState classWalkState, indexInParent int, funcMap walk.VisitorFuncMap[classWalkState]) classWalkState {
		classWalkState.IsExport = true

		decorator := node.ChildByFieldName("decorator")
		if decorator != nil {
			classWalkState.Decorator = decorator
		}

		classWalkState = walk.VisitNamedChildren(node, classWalkState, funcMap)

		classWalkState.IsExport = false
		classWalkState.Decorator = nil

		return classWalkState
	}

	classVisitor := func(node *sitter.Node, classWalkState classWalkState, indexInParent int, funcMap walk.VisitorFuncMap[classWalkState]) classWalkState {
		classContent := node.Content([]byte(file.Content))

		var class *Class

		_, err := utils.ParseFile(false, classContent, utils.TypeScript, nil,
			func(classRoot *sitter.Node, content []byte, _ any) (any, error) {
				uri := file.URI
				className := ExtractClassName(classRoot, content)

				var found bool
				class, found = state.Classes[ClassId(uri, className)]
				if !found {
					c := NewClass(classContent, file, node)
					c.Name = className
					class = &c
				} else {
					class.Reset()
					class.Node = node
					class.Content = CStr2GoStr(content)
					class.Name = className
				}

				ExtractMetadata(class, classRoot, []byte(class.Content))

				err := ExtractTypeScriptDefinitions(class, classRoot, []byte(class.Content))
				if err != nil {
					return class, err
				}

				err = ExtractTypeScriptUsages(class, classRoot, content)
				if err != nil {
					return class, err
				}

				var templateFilename string

				if classWalkState.Decorator != nil {
					templateFilename, err = ExtractTemplateFilename(class, classWalkState.Decorator, []byte(file.Content))
				} else {
					templateFilename, err = ExtractTemplateFilename(class, classRoot, content)
				}

				if err != nil {
					return class, err
				}

				if templateFilename != "" {
					err = handleTemplate(state, class, templateFilename)
					if err != nil {
						return class, err
					}
				}

				return nil, nil
			})

		if err != nil || class == nil {
			return classWalkState
		}

		file.Classes = append(file.Classes, class)
		if classWalkState.IsExport {
			export := Reference{Node: node, Name: class.Name, Class: class}
			file.Exports = append(file.Exports, &export)
		}

		return classWalkState
	}

	funcMap["abstract_class_declaration"] = classVisitor
	funcMap["class_declaration"] = classVisitor
	funcMap["interface_declaration"] = classVisitor

	classWalkState := classWalkState{}

	walk.Walk(root, classWalkState, funcMap)
}

func IndexTypeScriptFileFromIndexer(state *State, filename string) error {
	var err error

	file, err := createFileIfNotExists(state, filename, "", 0)
	if err != nil {
		return err
	}
	file.ResetClasses()

	_, err = utils.ParseFile(false, file.Content, utils.TypeScript, nil, func(root *sitter.Node, fileContent []byte, _ any) (any, error) {
		err = extractFileImports(root, file)
		if err != nil {
			return nil, err
		}

		parseClasses(state, root, file)

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
	file.ResetClasses()

	_, err = utils.ParseFile(false, file.Content, utils.TypeScript, nil, func(root *sitter.Node, fileContent []byte, _ any) (any, error) {
		err = extractFileImports(root, file) // todo need to reset imports too
		if err != nil {
			return nil, err
		}

		parseClasses(state, root, file)

		return nil, nil
	})

	return err
}

func handleTemplate(state *State, class *Class, templateFilename string) error {
	return IndexPugFromTypeScript(state, class, templateFilename)
}

// Must not store references to any nodes on the Class. This can be called on either file-based or class-based content
func ExtractTemplateFilename(class *Class, root *sitter.Node, content []byte) (string, error) {
	funcMap := walk.NewVisitorFuncsMap[typescriptWalkState]()
	funcMap["decorator"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		call := node.NamedChild(0)
		if call.Type() != "call_expression" {
			return state
		}

		decoratorNameNode := call.ChildByFieldName("function")
		if decoratorNameNode == nil {
			return state
		}

		decoratorName := decoratorNameNode.Content(content)
		if decoratorName != "Component" {
			return state
		}

		state.InDecorator = true

		for i := range node.NamedChildCount() {
			index := int(i)
			state = walk.VisitNode(node.NamedChild(index), state, index, funcMap)
		}

		state.InDecorator = false

		return state
	}

	funcMap["pair"] = func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		if !state.InDecorator {
			for i := range node.NamedChildCount() {
				index := int(i)
				state = walk.VisitNode(node.NamedChild(index), state, index, funcMap)
			}

			return state
		}

		keyNode := node.ChildByFieldName("key")
		if keyNode == nil {
			return state
		}

		if keyNode.Content(content) != "templateUrl" {
			return state
		}

		valueNode := node.ChildByFieldName("value")
		if valueNode == nil {
			return state
		}

		relativeTemplatePathNode := valueNode.NamedChild(0)
		if relativeTemplatePathNode == nil {
			return state
		}

		relativeTemplatePath := relativeTemplatePathNode.Content(content)
		if relativeTemplatePath == "" {
			return state
		}

		controllerDirectory := filepath.Dir(class.File.Filename())

		templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativeTemplatePath))
		if err != nil {
			return state
		}

		if utils.FileExists(templateFilePath) {
			state.TemplateFilename = templateFilePath
			return state
		}

		return state
	}

	s := typescriptWalkState{InDecorator: false, Class: class}
	s = walk.Walk(root, s, funcMap)

	return s.TemplateFilename, nil
}

func ExtractClassName(root *sitter.Node, content []byte) string {
	funcMap := walk.NewVisitorFuncsMap[string]()

	classVisitor := func(node *sitter.Node, state string, indexInParent int, funcMap walk.VisitorFuncMap[string]) string {
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return state
		}

		return nameNode.Content(content)
	}

	funcMap["abstract_class_declaration"] = classVisitor
	funcMap["class_declaration"] = classVisitor
	funcMap["interface_declaration"] = classVisitor

	return walk.Walk(root, "", funcMap)
}

func ExtractMetadata(class *Class, root *sitter.Node, content []byte) {
	funcMap := walk.NewVisitorFuncsMap[*Class]()

	classVisitor := func(node *sitter.Node, state *Class, indexInParent int, funcMap walk.VisitorFuncMap[*Class]) *Class {
		for i := range node.NamedChildCount() {
			child := node.NamedChild(int(i))
			t := child.Type()

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

					state.ExtendsIdentNames = extendsIdentifiers
				} else if jt == "implements_clause" {
					implementsClause := clause
					identCount := int(implementsClause.NamedChildCount())
					implementsIdentifiers := make([]string, identCount)

					for i := range identCount {
						implementsIdentifiers[i] = implementsClause.NamedChild(i).Content(content)
					}

					state.ImplementsIdentNames = implementsIdentifiers
				}
			}

		}

		return nil
	}

	funcMap["abstract_class_declaration"] = classVisitor
	funcMap["class_declaration"] = classVisitor
	funcMap["interface_declaration"] = classVisitor

	walk.Walk(root, class, funcMap)
}

func ExtractTypeScriptUsages(class *Class, root *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[typescriptWalkState]()
	funcMap["member_expression"] = visitUsageExpression(content)
	funcMap["subscript_expression"] = visitUsageExpression(content)

	s := typescriptWalkState{Class: class}
	walk.Walk(root, s, funcMap)

	return nil
}

func ExtractTypeScriptDefinitions(class *Class, root *sitter.Node, content []byte) error {
	funcMap := walk.NewVisitorFuncsMap[typescriptWalkState]()
	funcMap["method_definition"] = visitDefinition(content)
	funcMap["method_signature"] = visitDefinition(content)
	funcMap["property_definition"] = visitDefinition(content)
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
	s = walk.Walk(root, s, funcMap)

	return nil
}

func addUsage(class *Class, name string, node *sitter.Node, content []byte) {
	access := LocalAccess
	if isInConstructor(node, content) {
		access = ConstructorAccess
	}

	usageInstance := UsageInstance{access, node}

	class.SetUsageAccessType(name, usageInstance.Access)
	class.AppendUsage(name, &usageInstance)
	class.AppendDefinitionUsage(name, &usageInstance)
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

func handleDecorator(node *sitter.Node, content []byte) Decorator {
	functionExpression := node.NamedChild(0)
	decoratorNameNode := functionExpression.ChildByFieldName("function")

	var decoratorName string
	if decoratorNameNode != nil { // @Decorator()
		decoratorName = decoratorNameNode.Content(content)
	} else { // @Decorator
		decoratorName = functionExpression.Content(content)
	}

	isAngularDecorator := IsAngularDecorator(decoratorName)
	return Decorator{isAngularDecorator, decoratorName}
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
					state = walk.VisitNode(node.NamedChild(index), state, index, funcMap)
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
					state = walk.VisitNode(node.NamedChild(index), state, index, funcMap)
				}

				return state
			}
		}

		varName := varNode.Content(content)
		addUsage(state.Class, varName, node, content)

		for i := range node.NamedChildCount() {
			index := int(i)
			state = walk.VisitNode(node.NamedChild(index), state, index, funcMap)
		}

		return state
	}
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
			Decorators:  []Decorator{},
			Node:        node,
			UsageAccess: NoAccess,
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

		for i := range node.ChildCount() {
			index := int(i)
			state = walk.VisitNode(node.Child(index), state, index, funcMap)
		}

		finalDefinition := state.DefinitionStack.Pop()
		state.Class.AddDefinition(*finalDefinition)

		return state
	}
}
