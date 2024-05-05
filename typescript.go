package main

import (
	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTypeScriptUsages(state State, root *sitter.Node, content []byte) (State, error) {
	state, _ = WithCaptures(QueryPrototypeUsage, TypeScript, content, state, HandleCapture[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[2].Node
		name := node.Content(content)

		state = addUsage(state, name, node, content)

		return state, nil
	}))

	return WithCaptures(QueryPropertyUsage, TypeScript, content, state, HandleCapture[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[0].Node
		name := node.Content(content)

		state = addUsage(state, name, node, content)

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
