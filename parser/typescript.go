package parser

import (
	"path"
	"path/filepath"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type typescriptWalkState struct {
	DefinitionStack utils.Stack[Definition]
	InDecorator     bool
	File
}

func HandleTypeScriptFile(file File) (File, error) {
	fromDisk := file.Content == ""
	var source string
	if fromDisk {
		source = file.Filename()
	} else {
		source = file.Content
	}

	return utils.ParseFile(fromDisk, source, utils.TypeScript, file,
		func(root *sitter.Node, content []byte, file File) (File, error) {
			file = file.SetContent(CStr2GoStr(content))

			file, err := ExtractTypeScriptDefinitions(file, root, content)
			if err != nil {
				return file, err
			}

			file, err = ExtractTypeScriptUsages(file, root, content)
			if err != nil {
				return file, err
			}

			file, err = ExtractTemplateFilename(file, root, content)
			if err != nil {
				return file, err
			}

			return file, nil
		})
}

func ExtractTemplateFilename(file File, root *sitter.Node, content []byte) (File, error) {
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

		controllerDirectory := filepath.Dir(file.Filename())

		templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativeTemplatePath))
		if err != nil {
			return state
		}

		if utils.FileExists(templateFilePath) {
			state.Template = templateFilePath
			return state
		}

		return state
	}

	s := typescriptWalkState{InDecorator: false, File: file}
	s = walk.Walk(root, s, funcMap)

	return s.File, nil
}

func ExtractTypeScriptUsages(file File, root *sitter.Node, content []byte) (File, error) {
	funcMap := walk.NewVisitorFuncsMap[typescriptWalkState]()
	funcMap["member_expression"] = visitUsageExpression(content)
	funcMap["subscript_expression"] = visitUsageExpression(content)

	s := typescriptWalkState{File: file}
	s = walk.Walk(root, s, funcMap)

	return s.File, nil
}

func ExtractTypeScriptDefinitions(file File, root *sitter.Node, content []byte) (File, error) {
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

	s := typescriptWalkState{File: file}
	s = walk.Walk(root, s, funcMap)

	return s.File, nil
}

func addUsage(file File, name string, node *sitter.Node, content []byte) File {
	access := LocalAccess
	if isInConstructor(node, content) {
		access = ConstructorAccess
	}

	usageInstance := UsageInstance{access, node}

	file = file.SetUsageAccessType(name, usageInstance.Access)
	file = file.AppendUsage(name, usageInstance)
	file = file.AppendDefinitionUsage(name, usageInstance)

	return file
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
	decoratorName := node.Content(content)
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
				return state
			}
		}

		varNode := node.ChildByFieldName("property")
		if varNode == nil {
			varNode = node.ChildByFieldName("index")
			varNode = varNode.NamedChild(0)

			if varNode == nil || varNode.Type() != "string_fragment" {
				return state
			}
		}

		varName := varNode.Content(content)
		state.File = addUsage(state.File, varName, node, content)

		return state
	}
}

func visitDefinition(content []byte) walk.VisitorFunction[typescriptWalkState] {
	return func(node *sitter.Node, state typescriptWalkState, indexInParent int, funcMap walk.VisitorFuncMap[typescriptWalkState]) typescriptWalkState {
		state.DefinitionStack.Push(Definition{})
		state.DefinitionStack.Peek().Decorators = []Decorator{}
		state.DefinitionStack.Peek().UsageAccess = NoAccess

		state.DefinitionStack.Peek().Node = node

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

		for i := range node.NamedChildCount() {
			index := int(i)
			state = walk.VisitNode(node.NamedChild(index), state, index, funcMap)
		}

		definition := state.DefinitionStack.Pop()
		state.File.AddDefinition(definition.Name, *definition)

		return state
	}
}
