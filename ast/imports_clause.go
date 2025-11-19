package ast

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func doExtractImports(node *sitter.Node, content []byte) ([]*ImportParseResult, error) {
	funcMap := walk.NewVisitorFuncsMap[[]*ImportParseResult]()
	funcMap["import_statement"] = func(node *sitter.Node, state []*ImportParseResult, indexInParent int, funcMap walk.VisitorFuncMap[[]*ImportParseResult]) []*ImportParseResult {
		isType := false

		importNode := node

		for i := range node.ChildCount() {
			child := node.Child(int(i))
			if child.Type() == "type" {
				isType = true
				break
			}
		}

		internalFuncMap := walk.NewVisitorFuncsMap[*ImportParseResult]()
		internalFuncMap["import_clause"] = func(node *sitter.Node, state *ImportParseResult, indexInParent int, internalFuncMap walk.VisitorFuncMap[*ImportParseResult]) *ImportParseResult {
			state.Clause = node

			for i := range node.ChildCount() {
				walk.VisitNode(node.Child(int(i)), state, int(i), internalFuncMap)
			}

			return state
		}

		internalFuncMap["import_specifier"] = func(node *sitter.Node, state *ImportParseResult, indexInParent int, internalFuncMap walk.VisitorFuncMap[*ImportParseResult]) *ImportParseResult {
			state.Import = importNode

			localIsType := isType
			var nameNode *sitter.Node
			var aliasNode *sitter.Node

			for i := range node.ChildCount() {
				child := node.Child(int(i))
				if child.Type() == "type" {
					localIsType = true
				}

				if node.FieldNameForChild(int(i)) == "name" {
					nameNode = child
				}

				if node.FieldNameForChild(int(i)) == "alias" {
					aliasNode = child
				}
			}

			if nameNode == nil {
				return state
			}

			if aliasNode == nil {
				aliasNode = nameNode
			}

			name := nameNode.Content(content)
			alias := aliasNode.Content(content)

			imp := ImportIdentifier{
				ForeignIdentifier: name,
				IsType:            localIsType,
				LocalIdentifier:   alias,
			}
			state.Imports = append(state.Imports, imp)

			return state
		}

		importParseResult := ImportParseResult{IsType: isType}

		packageStringNode := node.ChildByFieldName("source")
		if packageStringNode != nil {
			packageNode := packageStringNode.NamedChild(0)
			if packageNode != nil {
				importParseResult.Package = packageNode.Content(content)
			}
		}

		walk.Walk(node, &importParseResult, internalFuncMap)

		return append(state, &importParseResult)
	}

	return walk.Walk(node, []*ImportParseResult{}, funcMap), nil
}

func ExtractImports(node *sitter.Node, content []byte) ([]*ImportParseResult, error) {
	if node != nil {
		return doExtractImports(node, content)
	}

	return utils.ParseFile(false, CStr2GoStr(content), utils.TypeScript, []*ImportParseResult{}, func(root *sitter.Node, content []byte, state []*ImportParseResult) ([]*ImportParseResult, error) {
		return doExtractImports(root, content)
	})
}

func FindPackageImport(importResults []*ImportParseResult, packageName string, isType bool) *ImportParseResult {
	i, found := findPackageFromResults(packageName, isType, importResults)
	if !found {
		return nil
	}

	return i
}

func AddToImport(importResults []*ImportParseResult, packageName string, toAdd []string, isType bool) utils.TextEdits {
	if len(toAdd) == 0 {
		return utils.TextEdits{}
	}

	importResult := FindPackageImport(importResults, packageName, isType)

	if importResult == nil {
		slices.SortFunc(importResults, func(a *ImportParseResult, b *ImportParseResult) int {
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
			packageIsRelative := strings.HasPrefix(packageName, ".")

			for index, result := range importResults {
				resultIsRelative := strings.HasPrefix(result.Package, ".")

				if packageIsRelative && !resultIsRelative {
					continue
				}

				r := result

				if (!packageIsRelative && resultIsRelative) || (result.Package >= packageName) {
					if !packageIsRelative && resultIsRelative {
						r = importResults[index-1]
					}

					lastPoint := r.Import.EndPoint()
					editRange = utils.Range{Start: utils.PositionFromPoint(lastPoint), End: utils.PositionFromPoint(lastPoint)}
					text = "\n" + text

					return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
				}
			}

			lastPoint := importResults[len(importResults)-1].Import.EndPoint()
			editRange = utils.Range{Start: utils.PositionFromPoint(lastPoint), End: utils.PositionFromPoint(lastPoint)}
			text = "\n" + text
		}

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	hasAdded := false
	for _, add := range toAdd {
		if !slices.ContainsFunc(importResult.Imports, func(i ImportIdentifier) bool { return i.ForeignIdentifier == add }) {
			(*importResult).Imports = append(importResult.Imports, ImportIdentifier{ForeignIdentifier: add, LocalIdentifier: add, IsType: isType})
			hasAdded = true
		}
	}

	if hasAdded {
		slices.SortFunc((*importResult).Imports, func(a ImportIdentifier, b ImportIdentifier) int {
			if a.ForeignIdentifier > b.ForeignIdentifier {
				return 1
			} else if a.ForeignIdentifier < b.ForeignIdentifier {
				return -1
			} else {
				return 0
			}
		})

		buildImportString := func(i ImportIdentifier) string {
			if i.ForeignIdentifier == i.LocalIdentifier {
				return i.ForeignIdentifier
			}

			return fmt.Sprintf("%s as %s", i.ForeignIdentifier, i.LocalIdentifier)
		}

		importStrings := make([]string, len(importResult.Imports))
		for index, i := range importResult.Imports {
			importStrings[index] = buildImportString(i)
		}

		text := "{" + strings.Join(importStrings, ", ") + "}"

		node := importResult.Clause
		editRange := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: text}}
	}

	return nil
}

// Should handle type imports
func AddImportToFile(content []byte, packageName string, toAdd []string, toAddTypes []string) (utils.TextEdits, error) {
	edits := utils.TextEdits{}

	importResults, err := ExtractImports(nil, content)
	if err != nil {
		return edits, err
	}

	importEdits := AddToImport(importResults, packageName, toAdd, false)
	for _, edit := range AddToImport(importResults, packageName, toAddTypes, true) {
		importEdits = append(importEdits, edit)
	}

	return importEdits, nil
}

func findPackageFromResults(packageName string, isType bool, results []*ImportParseResult) (*ImportParseResult, bool) {
	for _, result := range results {
		if result.Package == packageName && result.IsType == isType {
			return result, true
		}
	}

	return nil, false
}

type ImportIdentifier struct {
	ForeignIdentifier string
	IsType            bool
	LocalIdentifier   string
}

type ImportParseResult struct {
	Clause  *sitter.Node
	Import  *sitter.Node
	Imports []ImportIdentifier
	IsType  bool
	Package string
}

func CStr2GoStr(b []byte) string {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		i = len(b)
	}

	return string(b[:i])
}
