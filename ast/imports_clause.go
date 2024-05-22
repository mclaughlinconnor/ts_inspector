package ast

import (
	"fmt"
	"slices"
	"strings"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractImports(content []byte) (map[string]ImportParseResult, error) {
	result, err := utils.WithMatches(utils.QueryImport, utils.TypeScript, content, map[string]ImportParseResult{}, func(captures utils.Captures, returnValue map[string]ImportParseResult) (map[string]ImportParseResult, error) {
		var packageName string
		if captures["package"] != nil {
			packageName = captures["package"][0].Content(content)
		}

		var importResult ImportParseResult
		p, found := returnValue[packageName]
		if found {
			importResult = p
		} else {
			importResult = ImportParseResult{}
		}

		if captures["clause"] != nil {
			importResult.Clause = captures["clause"][0]
		}

		if captures["identifier"] != nil {
			for _, identifier := range captures["identifier"] {
				importResult.Imports = append(importResult.Imports, identifier.Content(content))
			}
		}

		returnValue[packageName] = importResult

		return returnValue, nil
	})

	return result, err
}

func FindPackageImport(importResults Imports, packageName string) *ImportParseResult {
	i, found := importResults[packageName]
	if !found {
		return nil
	}

	return &i
}

func AddToImport(importResults Imports, packageName string, toAdd string) utils.TextEdits {
	importResult := FindPackageImport(importResults, packageName)

	if importResult == nil {
		var max uint32 = 0
		var maxKey string
		for key, i := range importResults {
			if i.Clause.EndByte() > max {
				max = i.Clause.EndByte()
				maxKey = key
			}
		}

		text := fmt.Sprintf("import {%s} from '%s'", toAdd, packageName)

		var editRange utils.Range
		if maxKey != "" {
			lastPoint := importResults[maxKey].Clause.EndPoint()
			editRange = utils.Range{Start: utils.PositionFromPoint(lastPoint), End: utils.PositionFromPoint(lastPoint)}
		} else {
			position := utils.Position{Line: 0, Character: 0}
			editRange = utils.Range{Start: position, End: position}
			text = text + "\n\n"
		}

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	if !slices.Contains(importResult.Imports, toAdd) {
		importResult.Imports = append(importResult.Imports, toAdd)
		slices.Sort(importResult.Imports)
		text := "{" + strings.Join(importResult.Imports, ", ") + "}"

		node := importResult.Clause
		editRange := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	return nil
}

// Should handle type imports
func AddImportToFile(content []byte, packageName string, toAdd string) (utils.TextEdits, error) {
	edits := utils.TextEdits{}

	importResults, err := ExtractImports(content)
	if err != nil {
		return edits, err
	}

	importEdits := AddToImport(importResults, packageName, toAdd)

	return importEdits, nil

}

type ImportParseResult struct {
	Clause  *sitter.Node
	Imports []string
	Package string
}

type Imports map[string]ImportParseResult
