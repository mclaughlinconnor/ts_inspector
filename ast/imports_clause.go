package ast

import (
	"fmt"
	"slices"
	"strings"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractImports(content []byte) ([]ImportParseResult, error) {
	result, err := utils.WithMatches(utils.QueryImport, utils.TypeScript, content, []ImportParseResult{}, func(captures utils.Captures, returnValue []ImportParseResult) ([]ImportParseResult, error) {
		var importResult ImportParseResult

		if captures["import"] != nil {
			importResult.Import = captures["import"][0]
		}

		if captures["package"] != nil {
			importResult.Package = captures["package"][0].Content(content)
		}

		if captures["clause"] != nil {
			importResult.Clause = captures["clause"][0]
		}

		if captures["type"] != nil {
			importResult.IsType = true
		}

		// Can't get a (named_imports (import_specifier)* @specifier) to work at all
		if captures["named_imports"] != nil {
			node := captures["named_imports"][0]

			child := node.Child(0)

			for child != nil {
				import_specifier_node := child.Child(0)
				for import_specifier_node != nil {
					if import_specifier_node.Type() == "identifier" {
						importResult.Imports = append(importResult.Imports, import_specifier_node.Content(content))
					}

					import_specifier_node = import_specifier_node.NextNamedSibling()
				}

				child = child.NextNamedSibling()
			}
		}

		return append(returnValue, importResult), nil
	})

	return result, err
}

func FindPackageImport(importResults []ImportParseResult, packageName string, isType bool) *ImportParseResult {
	i, found := findPackageFromResults(packageName, isType, importResults)
	if !found {
		return nil
	}

	return i
}

func AddToImport(importResults []ImportParseResult, packageName string, toAdd []string, isType bool) utils.TextEdits {
	if len(toAdd) == 0 {
		return utils.TextEdits{}
	}

	importResult := FindPackageImport(importResults, packageName, isType)

	if importResult == nil {
		slices.SortFunc(importResults, func(a ImportParseResult, b ImportParseResult) int {
			return int(a.Import.EndByte()) - int(b.Import.EndByte())
		})

		var text string
		if isType {
			text = fmt.Sprintf("import type {%s} from '%s'", strings.Join(toAdd, ", "), packageName)
		} else {
			text = fmt.Sprintf("import {%s} from '%s'", strings.Join(toAdd, ", "), packageName)
		}

		var editRange utils.Range
		if len(importResults) == 0 {
			position := utils.Position{Line: 0, Character: 0}
			editRange = utils.Range{Start: position, End: position}
			text = text + "\n\n"
		} else {
			lastPoint := importResults[len(importResults)-1].Import.EndPoint()
			editRange = utils.Range{Start: utils.PositionFromPoint(lastPoint), End: utils.PositionFromPoint(lastPoint)}
			text = "\n" + text
		}

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	hasAdded := false
	for _, add := range toAdd {
		if !slices.Contains(importResult.Imports, add) {
			(*importResult).Imports = append(importResult.Imports, add)
			hasAdded = true
		}
	}

	if hasAdded {
		slices.Sort((*importResult).Imports)
		text := "{" + strings.Join((*importResult).Imports, ", ") + "}"

		node := importResult.Clause
		editRange := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	return nil
}

// Should handle type imports
func AddImportToFile(content []byte, packageName string, toAdd []string, toAddTypes []string) (utils.TextEdits, error) {
	edits := utils.TextEdits{}

	importResults, err := ExtractImports(content)
	if err != nil {
		return edits, err
	}

	importEdits := AddToImport(importResults, packageName, toAdd, false)
	for _, edit := range AddToImport(importResults, packageName, toAddTypes, true) {
		importEdits = append(importEdits, edit)
	}

	return importEdits, nil
}

func findPackageFromResults(packageName string, isType bool, results []ImportParseResult) (*ImportParseResult, bool) {
	for _, result := range results {
		if result.Package == packageName && result.IsType == isType {
			return &result, true
		}
	}

	return nil, false
}

type ImportParseResult struct {
	Clause  *sitter.Node
	Import  *sitter.Node
	Imports []string
	IsType  bool
	Package string
}
