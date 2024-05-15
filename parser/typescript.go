package parser

import (
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTypeScriptUsages(file File, state State, root *sitter.Node, content []byte) (State, error) {
	state, _ = WithMatches(QueryPrototypeUsage, TypeScript, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[2].Node
		name := node.Content(content)

		returnValue = addUsage(file, returnValue, name, node, content)

		return returnValue, nil
	}))

	return WithMatches(QueryPropertyUsage, TypeScript, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[0].Node
		name := node.Content(content)

		returnValue = addUsage(file, returnValue, name, node, content)

		return returnValue, nil
	}))
}

func ExtractTypeScriptDefinitions(file File, state State, root *sitter.Node, content []byte) (State, error) {
	state, _ = WithMatches(QueryMethodDefinition, TypeScript, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		definition := Definition{}
		definition.Decorators = []Decorator{}

		for _, capture := range captures {
			definition = handleDefinition(definition, capture.Node, content)
		}

		returnValue[file.Filename()].AddDefinition(definition.Name, definition)

		return returnValue, nil
	}))

	return WithMatches(QueryPropertyDefinition, TypeScript, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		var definitionNode *sitter.Node
		var accessibilityNode *sitter.Node
		var nameNode *sitter.Node

		nodeIndex := 0
		definitionNode = captures[nodeIndex].Node
		nodeIndex++

		decorators, nodeIndex := handleDecorators(captures, nodeIndex, content)

		accessibilityNode = captures[nodeIndex].Node
		nodeIndex++
		nameNode = captures[nodeIndex].Node
		nodeIndex++

		name := nameNode.Content(content)
		accessibility := accessibilityNode.Content(content)

		a, err := CalculateAccessibilityFromString(accessibility)
		if err != nil {
			log.Fatal(err)
		}

		returnValue[file.Filename()] = returnValue[file.Filename()].AddDefinition(name, CreatePropertyDefinition(a, decorators, name, definitionNode))

		return returnValue, nil
	}))
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

func handleDefinition(definition Definition, node *sitter.Node, content []byte) Definition {
	if node.Type() == "identifier" { // Should be safer. identifier doesn't just mean identifiers inside decorators
		definition.Decorators = append(definition.Decorators, handleDecorator(node, content))
	} else if node.Type() == "accessibility_modifier" {
		a, err := CalculateAccessibilityFromString(node.Content(content))
		if err != nil {
			log.Fatal(err)
		}
		definition.AccessModifier = a
	} else if node.Type() == "static" {
		definition.Static = true
	} else if node.Type() == "override_modifier" {
		definition.Override = true
	} else if node.Type() == "readonly" {
		definition.Readonly = true
	} else if node.Type() == "async" {
		definition.Async = true
	} else if node.Type() == "*" {
		definition.Generator = true
	} else if node.Type() == "set" {
		definition.Setter = true
	} else if node.Type() == "get" {
		definition.Getter = true
	} else if node.Type() == "property_identifier" {
		definition.Name = node.Content(content)
	} else if node.Type() == "method_definition" {
		definition.Node = node
	} else {
		log.Println(node.Type())
	}

	return definition
}

func handleDecorators(captures []sitter.QueryCapture, startIndex int, content []byte) ([]Decorator, int) {
	decorators := []Decorator{}
	nodeIndex := startIndex

	for captures[nodeIndex].Node.Type() == "identifier" {
		decorators = append(decorators, handleDecorator(captures[nodeIndex].Node, content))
		nodeIndex++
	}

	return decorators, nodeIndex
}

func handleDecorator(node *sitter.Node, content []byte) Decorator {
	decoratorName := node.Content(content)
	isAngularDecorator := IsAngularDecorator(decoratorName)
	return Decorator{isAngularDecorator, decoratorName}
}
