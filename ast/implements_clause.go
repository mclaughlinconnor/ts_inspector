package ast

import (
	"slices"
	"strings"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractImplements(content []byte) (ImplementParseResult, error) {
	result, err := utils.WithMatches(utils.QueryClassImplements, utils.TypeScript, content, ImplementParseResult{nil, []string{}}, func(captures utils.Captures, returnValue ImplementParseResult) (ImplementParseResult, error) {
		if captures["clause"] != nil {
			returnValue.Clause = captures["clause"][0]
		}

		if captures["implements"] != nil {
			for _, identifier := range captures["implements"] {
				returnValue.Implements = append(returnValue.Implements, identifier.Content(content))
			}
		}

		return returnValue, nil
	})

	return result, err
}

func AddToImplement(implementResult ImplementParseResult, toAdd string) utils.TextEdits {
	if !slices.Contains(implementResult.Implements, toAdd) {
		implementResult.Implements = append(implementResult.Implements, toAdd)
		slices.Sort(implementResult.Implements)
		text := strings.Join(implementResult.Implements, ", ")

		node := implementResult.Clause

		editRange := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}
		editRange.Start.Character = editRange.Start.Character + uint32(len("implements "))

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	return nil
}

// Should handle type implements
func AddImplementToFile(content []byte, toAdd string) (utils.TextEdits, error) {
	implementResult, err := ExtractImplements(content)
	if err != nil {
		return nil, err
	}

	implementEdits := AddToImplement(implementResult, toAdd)

	return implementEdits, nil
}

type ImplementParseResult struct {
	Clause     *sitter.Node
	Implements []string
}

type Implements map[string]ImplementParseResult
