package parser

import (
	"fmt"
	"path"
	"path/filepath"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

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
	return utils.WithMatches(utils.QueryComponentDecorator, utils.TypeScript, content, file, func(captures utils.Captures, returnValue File) (File, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		relativeTemplatePath := captures["template"][0].Content(content)
		controllerDirectory := filepath.Dir(file.Filename())

		templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativeTemplatePath))
		if err != nil {
			return returnValue, err
		}

		if utils.FileExists(templateFilePath) {
			returnValue.Template = templateFilePath
			return returnValue, nil
		}

		return returnValue, fmt.Errorf("Unexpected template file does not exist: %s", relativeTemplatePath)
	})
}

func ExtractTypeScriptUsages(file File, root *sitter.Node, content []byte) (File, error) {
	file, _ = utils.WithMatches(utils.QueryPrototypeUsage, utils.TypeScript, content, file, func(captures utils.Captures, returnValue File) (File, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures["var"][0]
		name := node.Content(content)

		returnValue = addUsage(returnValue, name, node, content)

		return returnValue, nil
	})

	return utils.WithMatches(utils.QueryPropertyUsage, utils.TypeScript, content, file, func(captures utils.Captures, returnValue File) (File, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures["var"][0]
		name := node.Content(content)

		returnValue = addUsage(returnValue, name, node, content)

		return returnValue, nil
	})
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
