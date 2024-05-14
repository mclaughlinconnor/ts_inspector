package parser

import (
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTypeScriptUsages(file File, root *sitter.Node, content []byte) (File, error) {
	file, _ = WithMatches(QueryPrototypeUsage, TypeScript, content, file, HandleMatch[File](func(captures []sitter.QueryCapture, returnValue File) (File, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[2].Node
		name := node.Content(content)

		file = addUsage(file, name, node, content)

		return file, nil
	}))

	return WithMatches(QueryPropertyUsage, TypeScript, content, file, HandleMatch[File](func(captures []sitter.QueryCapture, returnValue File) (File, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[0].Node
		name := node.Content(content)

		file = addUsage(file, name, node, content)

		return file, nil
	}))
}

func ExtractTypeScriptDefinitions(file File, root *sitter.Node, content []byte) (File, error) {
	file, _ = WithMatches(QueryMethodDefinition, TypeScript, content, file, HandleMatch[File](func(captures []sitter.QueryCapture, returnValue File) (File, error) {
		definition := Definition{}
		definition.Decorators = []Decorator{}

		for _, capture := range captures {
			definition = handleDefinition(definition, capture.Node, content)
		}

		file.Definitions[definition.Name] = definition

		return returnValue, nil
	}))

	return WithMatches(QueryPropertyDefinition, TypeScript, content, file, HandleMatch[File](func(captures []sitter.QueryCapture, returnValue File) (File, error) {
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

		file.Definitions[name] = CreatePropertyDefinition(a, decorators, name, definitionNode)

		return file, nil
	}))
}

func addUsage(file File, name string, node *sitter.Node, content []byte) File {
	access := LocalAccess
	if isInConstructor(node, content) {
		access = ConstructorAccess
	}

	usageInstance := UsageInstance{access, node}

	usage, ok := file.Usages[name]
	if ok {
		existingUsages := usage
		existingUsages.Access = CalculateNewAccessType(existingUsages.Access, usageInstance.Access)
		existingUsages.Usages = append(existingUsages.Usages, usageInstance)
		file.Usages[name] = existingUsages
	} else {
		file.Usages[name] = Usage{
			usageInstance.Access,
			name,
			[]UsageInstance{usageInstance},
		}
	}

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
