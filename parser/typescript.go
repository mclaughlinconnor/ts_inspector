package parser

import (
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTypeScriptUsages(file File, state State, root *sitter.Node, content []byte) (State, error) {
	state, _ = utils.WithMatches(utils.QueryPrototypeUsage, utils.TypeScript, content, state, func(captures utils.Captures, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures["var"][0]
		name := node.Content(content)

		returnValue = addUsage(file, returnValue, name, node, content)

		return returnValue, nil
	})

	return utils.WithMatches(utils.QueryPropertyUsage, utils.TypeScript, content, state, func(captures utils.Captures, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures["var"][0]
		name := node.Content(content)

		returnValue = addUsage(file, returnValue, name, node, content)

		return returnValue, nil
	})
}

func ExtractTypeScriptDefinitions(file File, state State, root *sitter.Node, content []byte) (State, error) {
	state, _ = utils.WithMatches(utils.QueryMethodDefinition, utils.TypeScript, content, state, func(captures utils.Captures, returnValue State) (State, error) {
		definition := Definition{}
		definition.Decorators = []Decorator{}
		definition.UsageAccess = NoAccess
		var err error

		definition, err = handleDefinition(definition, captures, content)
		if err != nil {
			return returnValue, err
		}

		returnValue[file.Filename()].AddDefinition(definition.Name, definition)

		return returnValue, nil
	})

	return utils.WithMatches(utils.QueryPropertyDefinition, utils.TypeScript, content, state, func(captures utils.Captures, returnValue State) (State, error) {
		definitionNode := captures["definition"][0]
		name := captures["var"][0].Content(content)
		accessibility := captures["accessibility_modifier"][0].Content(content)

		decorators := handleDecorators(captures, content)

		a, err := CalculateAccessibilityFromString(accessibility)
		if err != nil {
			return returnValue, err
		}

		returnValue[file.Filename()] = returnValue[file.Filename()].AddDefinition(name, CreatePropertyDefinition(a, decorators, name, definitionNode))

		return returnValue, nil
	})
}

func addUsage(file File, state State, name string, node *sitter.Node, content []byte) State {
	access := LocalAccess
	if isInConstructor(node, content) {
		access = ConstructorAccess
	}

	file = state[file.Filename()]
	usageInstance := UsageInstance{access, node}

	file = file.SetUsageAccessType(name, usageInstance.Access)
	file = file.AppendUsage(name, usageInstance)
	file = file.AppendDefinitionUsage(name, usageInstance)

	state[file.Filename()] = file

	return state
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
