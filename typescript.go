package main

import (
	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractUsages(usages map[string]Usage, root *sitter.Node, content []byte) (returnedUsages map[string]Usage, err error) {
	qc, q := GetQuery(QueryPropertyUsage, TypeScript)

	qc.Exec(q, root)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		if len(m.Captures) == 0 {
			continue
		}

		node := m.Captures[0].Node
		name := node.Content(content)

		usageInstance := UsageInstance{LocalAccess, node}

		_, ok = usages[name]
		if ok {
      existingUsages := usages[name]
      existingUsages.Usages = append(existingUsages.Usages, usageInstance)
			usages[name] = existingUsages
		} else {
			usages[name] = Usage{
        LocalAccess,
        name,
        []UsageInstance{usageInstance},
      }
    }
	}
	return usages, nil
}
