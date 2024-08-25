package parser

import (
	"path"
	"path/filepath"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type typescriptWalkState struct {
	InDecorator bool
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
	file, _ = utils.WithMatches(utils.QueryMethodDefinition, utils.TypeScript, content, file, func(captures utils.Captures, returnValue File) (File, error) {
		definition := Definition{}
		definition.Decorators = []Decorator{}
		definition.UsageAccess = NoAccess
		var err error

		definition, err = handleDefinition(definition, captures, content)
		if err != nil {
			return returnValue, err
		}

		returnValue = returnValue.AddDefinition(definition.Name, definition)

		return returnValue, nil
	})

	return utils.WithMatches(utils.QueryPropertyDefinition, utils.TypeScript, content, file, func(captures utils.Captures, returnValue File) (File, error) {
		definitionNode := captures["definition"][0]
		name := captures["var"][0].Content(content)
		accessibility := captures["accessibility_modifier"][0].Content(content)

		decorators := handleDecorators(captures, content)

		a, err := CalculateAccessibilityFromString(accessibility)
		if err != nil {
			return returnValue, err
		}

		returnValue = returnValue.AddDefinition(name, CreatePropertyDefinition(a, decorators, name, definitionNode))

		return returnValue, nil
	})
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

func handleDefinition(definition Definition, captures utils.Captures, content []byte) (Definition, error) {
	if captures["definition"] != nil {
		definition.Node = captures["definition"][0]
	}

	if captures["decorator"] != nil {
		definition.Decorators = append(definition.Decorators, handleDecorator(captures["decorator"][0], content))
	}

	if captures["accessibility_modifier"] != nil {
		a, err := CalculateAccessibilityFromString(captures["accessibility_modifier"][0].Content(content))
		if err != nil {
			return definition, err
		}
		definition.AccessModifier = a
	}

	if captures["static"] != nil {
		definition.Static = true
	}

	if captures["override_modifier"] != nil {
		definition.Override = true
	}

	if captures["readonly"] != nil {
		definition.Readonly = true
	}

	if captures["async"] != nil {
		definition.Async = true
	}

	if captures["generator"] != nil {
		definition.Generator = true
	}

	if captures["name"] != nil {
		definition.Name = captures["name"][0].Content(content)
	}

	if captures["set"] != nil {
		definition.Setter = true
	}

	if captures["get"] != nil {
		definition.Getter = true
	}

	if captures["property_identifier"] != nil {
		definition.Name = captures["property_identifier"][0].Content(content)
	}

	if captures["method_definition"] != nil {
		definition.Node = captures["method_definition"][0]
	}

	return definition, nil
}

func handleDecorators(captures utils.Captures, content []byte) []Decorator {
	decorators := []Decorator{}

	for _, node := range captures["decorator"] {
		decorators = append(decorators, handleDecorator(node, content))
	}

	return decorators
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
