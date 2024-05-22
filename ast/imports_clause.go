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

func FindPackageImport(content []byte, packageName string) (ImportParseResult, bool, error) {
	importResult, err := ExtractImports(content)

	i, found := importResult[packageName]
	if !found {
		return ImportParseResult{}, false, err
	}

	return i, true, err
}

func AddToImport(importResult ImportParseResult, toAdd string) utils.TextEdits {
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

	importResult, found, err := FindPackageImport(content, packageName)
	if err != nil {
		return edits, err
	}

	if !found {
		// TODO: add handling for classes without an import
		return edits, fmt.Errorf("Could not find import clause")
	}

	importEdits := AddToImport(importResult, "OnInit")

	return importEdits, nil

}

type ImportParseResult struct {
	Clause  *sitter.Node
	Imports []string
	Package string
}

type Imports map[string]ImportParseResult
