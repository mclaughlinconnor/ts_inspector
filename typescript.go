package main

import (
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTypeScriptUsages(state State, root *sitter.Node, content []byte) (State, error) {
	state, _ = WithMatches(QueryPrototypeUsage, TypeScript, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[2].Node
		name := node.Content(content)

		state = addUsage(state, name, node, content)

		return state, nil
	}))

	return WithMatches(QueryPropertyUsage, TypeScript, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[0].Node
		name := node.Content(content)

		state = addUsage(state, name, node, content)

		return state, nil
	}))
}

func ExtractTypeScriptDefinitions(state State, root *sitter.Node, content []byte) (State, error) {
	return WithMatches(QueryPropertyDefinition, TypeScript, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		var definitionNode *sitter.Node
		var accessibilityNode *sitter.Node
		var nameNode *sitter.Node

		decoratorNodes := []*sitter.Node{}
		nodeIndex := 0

		definitionNode = captures[nodeIndex].Node
		nodeIndex++
		for captures[nodeIndex].Node.Type() == "identifier" {
			decoratorNodes = append(decoratorNodes, captures[nodeIndex].Node)
			nodeIndex++
		}

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

		decorators := []Decorator{}
		for _, decoratorNode := range decoratorNodes {
			decoratorName := decoratorNode.Content(content)
			isAngularDecorator := IsAngularDecorator(decoratorName)
			decorators = append(decorators, Decorator{isAngularDecorator, decoratorName})
		}

		state.Definitions[name] = Definition{a, decorators, name, definitionNode}

		return state, nil
	}))
}

func addUsage(state State, name string, node *sitter.Node, content []byte) State {
	access := LocalAccess
	if isInConstructor(node, content) {
		access = ConstructorAccess
	}

	usageInstance := UsageInstance{access, node}

	usage, ok := state.Usages[name]
	if ok {
		existingUsages := usage
		existingUsages.Access = CalculateNewAccessType(existingUsages.Access, usageInstance.Access)
		existingUsages.Usages = append(existingUsages.Usages, usageInstance)
		state.Usages[name] = existingUsages
	} else {
		state.Usages[name] = Usage{
			usageInstance.Access,
			name,
			[]UsageInstance{usageInstance},
		}
	}

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
