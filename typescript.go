package main

import (
	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTypeScriptUsages(usages Usages, root *sitter.Node, content []byte) (Usages, error) {
	usages, _ = WithCaptures(QueryPrototypeUsage, TypeScript, content, usages, HandleCapture[Usages](func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[2].Node
		name := node.Content(content)

		usages = addUsage(usages, name, node, content)

		return usages, nil
	}))

	return WithCaptures(QueryPropertyUsage, TypeScript, content, usages, HandleCapture[Usages](func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		node := captures[0].Node
		name := node.Content(content)

		usages = addUsage(usages, name, node, content)

		return usages, nil
	}))
}

func addUsage(usages Usages, name string, node *sitter.Node, content []byte) Usages {
	access := LocalAccess
	if isInConstructor(node, content) {
		access = ConstructorAccess
	}

	usageInstance := UsageInstance{access, node}

	_, ok := usages[name]
	if ok {
		existingUsages := usages[name]
		existingUsages.Access = CalculateNewAccessType(existingUsages.Access, usageInstance.Access)
		existingUsages.Usages = append(existingUsages.Usages, usageInstance)
		usages[name] = existingUsages
	} else {
		usages[name] = Usage{
			usageInstance.Access,
			name,
			[]UsageInstance{usageInstance},
		}
	}

	return usages
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
