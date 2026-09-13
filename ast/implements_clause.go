package ast

import (
	"slices"
	"strings"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func ExtractClassDefinition(content []byte) (*ClassParseResult, error) {
	result, err := utils.WithMatches(utils.QueryClassDefinition, utils.TypeScript, content, nil, func(captures utils.Captures, returnValue *ClassParseResult) (*ClassParseResult, error) {
		if returnValue == nil {
			returnValue = &ClassParseResult{}
		}

		if captures["name"] != nil {
			(*returnValue).NameNode = captures["name"][0]
		}

		if captures["type_parameters"] != nil {
			(*returnValue).TypeParameters = captures["type_parameters"][0]
		}

		if captures["extends_clause"] != nil {
			(*returnValue).ExtendsClause = captures["extends_clause"][0]
		}

		if captures["implements_clause"] != nil {
			(*returnValue).ImplementsClause = captures["implements_clause"][0]
		}

		if captures["identifier"] != nil {
			for _, identifier := range captures["identifier"] {
				(*returnValue).ImplementedIdentifiers = append((*returnValue).ImplementedIdentifiers, identifier.Utf8Text(content))
			}
		}

		return returnValue, nil
	})

	return result, err
}

func AddToImplement(classResult *ClassParseResult, toAdd string) utils.TextEdits {
	// There's no class to add anything to
	if classResult == nil {
		return nil
	}

	if classResult.ImplementsClause == nil {
		point := findImplementsInsertionPoint(classResult)
		editRange := utils.Range{Start: utils.LspPositionFromTsPosition(point), End: utils.LspPositionFromTsPosition(point)}
		text := " implements " + toAdd

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	if !slices.Contains(classResult.ImplementedIdentifiers, toAdd) {
		classResult.ImplementedIdentifiers = append(classResult.ImplementedIdentifiers, toAdd)
		slices.Sort(classResult.ImplementedIdentifiers)
		text := strings.Join(classResult.ImplementedIdentifiers, ", ")

		node := classResult.ImplementsClause

		editRange := utils.Range{Start: utils.LspPositionFromTsPosition(node.StartPosition()), End: utils.LspPositionFromTsPosition(node.EndPosition())}
		editRange.Start.Character = editRange.Start.Character + uint(len("implements "))

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	return nil
}

// Should handle type implements
func AddImplementToFile(content []byte, toAdd string) (utils.TextEdits, error) {
	implementResult, err := ExtractClassDefinition(content)
	if err != nil {
		return nil, err
	}

	implementEdits := AddToImplement(implementResult, toAdd)

	return implementEdits, nil
}

func findImplementsInsertionPoint(classResult *ClassParseResult) sitter.Point {
	if classResult.ExtendsClause != nil {
		return classResult.ExtendsClause.EndPosition()
	}

	if classResult.TypeParameters != nil {
		return classResult.TypeParameters.EndPosition()
	}

	return classResult.NameNode.EndPosition()
}

type ClassParseResult struct {
	ImplementedIdentifiers []string
	ImplementsClause       *sitter.Node
	ExtendsClause          *sitter.Node
	NameNode               *sitter.Node
	TypeParameters         *sitter.Node
}

type Implements map[string]ClassParseResult
