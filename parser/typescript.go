package parser

import (
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
	Class
}

type classWalkState struct {
	Classes []*Class
	Exports []*Export
}

func HandleTypeScriptFile(state *State, file File) (File, error) {
	fromDisk := file.Content == ""
	var source string
	if fromDisk {
		source = file.Filename()
	} else {
		source = file.Content
	}

	return utils.ParseFile(fromDisk, source, utils.TypeScript, file,
		func(root *sitter.Node, content []byte, file File) (File, error) {
			file.SetContent(CStr2GoStr(content))

			imports, err := ast.ExtractImports(root, content)
			if err != nil {
				return file, err
			}

			file.Imports = append(file.Imports, imports...)

			visitor := func(node *sitter.Node, classes classWalkState, indexInParent int, funcMap walk.VisitorFuncMap[classWalkState]) classWalkState {
				if node.Type() == "export_statement" {
					declaration := node.ChildByFieldName("declaration")
					if declaration != nil {
						if declaration.Type() != "class_declaration" {
							return classes
						}
					}
				}

				class := Class{Content: node.Content(content), File: &file, Node: node}

				class, err := utils.ParseFile(false, class.Content, utils.TypeScript, class,
					func(classRoot *sitter.Node, content []byte, class Class) (Class, error) {
						class, err := ExtractMetadata(&class, classRoot, []byte(class.Content))
						if err != nil {
							return class, err
						}

						class, err = ExtractTypeScriptDefinitions(class, classRoot, []byte(class.Content))
						if err != nil {
							return class, err
						}

						class, err = ExtractTypeScriptUsages(class, classRoot, content)
						if err != nil {
							return class, err
						}

						templateFilename, err := ExtractTemplateFilename(class, classRoot, content)
						if err != nil {
							return class, err
						}

						if templateFilename != "" {
							class, err = HandleTemplate(state, class, templateFilename)
							if err != nil {
								return class, err
							}
						}

						return class, nil
					})

				if err != nil {
					return classes
				}

				classes.Classes = append(classes.Classes, &class)

				if node.Type() == "export_statement" {
					export := Export{Node: node, Name: class.Name, Class: &class}
					classes.Exports = append(classes.Exports, &export)
				}

				return classes
			}

			funcMap := walk.NewVisitorFuncsMap[classWalkState]()

			funcMap["class_declaration"] = visitor
			funcMap["export_statement"] = visitor

			state := classWalkState{Classes: file.Classes, Exports: file.Exports}

			state = walk.Walk(root, state, funcMap)

			file.Classes = state.Classes
			file.Exports = state.Exports

			return file, nil
		})
}

func HandleTemplate(state *State, class Class, templateFilename string) (Class, error) {
	return HandlePugFile(state, class, templateFilename)
}

func ExtractTemplateFilename(class Class, root *sitter.Node, content []byte) (string, error) {
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

func ExtractMetadata(class *Class, root *sitter.Node, content []byte) (Class, error) {
	funcMap := walk.NewVisitorFuncsMap[*Class]()
	funcMap["class_declaration"] = func(node *sitter.Node, state *Class, indexInParent int, funcMap walk.VisitorFuncMap[*Class]) *Class {
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return state
		}

		state.Name = nameNode.Content(content)

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

		return state
	}

	walk.Walk(root, class, funcMap)

	return *class, nil
}

func ExtractTypeScriptUsages(class Class, root *sitter.Node, content []byte) (Class, error) {
	funcMap := walk.NewVisitorFuncsMap[typescriptWalkState]()
	funcMap["member_expression"] = visitUsageExpression(content)
	funcMap["subscript_expression"] = visitUsageExpression(content)

	s := typescriptWalkState{Class: class}
	s = walk.Walk(root, s, funcMap)

	return s.Class, nil
}

func ExtractTypeScriptDefinitions(class Class, root *sitter.Node, content []byte) (Class, error) {
	funcMap := walk.NewVisitorFuncsMap[typescriptWalkState]()
	funcMap["method_definition"] = visitDefinition(content)
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

	return s.Class, nil
}

func addUsage(class Class, name string, node *sitter.Node, content []byte) Class {
	access := LocalAccess
	if isInConstructor(node, content) {
		access = ConstructorAccess
	}

	usageInstance := UsageInstance{access, node}

	class = class.SetUsageAccessType(name, usageInstance.Access)
	class = class.AppendUsage(name, usageInstance)
	class = class.AppendDefinitionUsage(name, usageInstance)

	return class
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
		state.Class = addUsage(state.Class, varName, node, content)

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
		state.Class = state.Class.AddDefinition(*finalDefinition)

		return state
	}
}
