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
	funcMap["import_statement"] = func(node *sitter.Node, state []*ImportParseResult, indexInParent int, funcMap walk.VisitorFuncMap[[]*ImportParseResult]) ([]*ImportParseResult, error) {
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
		internalFuncMap["import_clause"] = func(node *sitter.Node, state *ImportParseResult, indexInParent int, internalFuncMap walk.VisitorFuncMap[*ImportParseResult]) (*ImportParseResult, error) {
			state.Clause = node

			for i := range node.ChildCount() {
				child := node.Child(int(i))
				if child.Type() == "identifier" {
					imp := ImportIdentifier{
						ForeignIdentifier: child.Content(content),
						IsType:            isType,
						LocalIdentifier:   child.Content(content),
					}

					state.Imports = append(state.Imports, imp)
					state.Import = importNode
				} else {
					_, err := walk.VisitNode(node.Child(int(i)), state, int(i), internalFuncMap, false)
					if err != nil {
						return nil, err
					}
				}
			}

			return state, nil
		}

		internalFuncMap["import_specifier"] = func(node *sitter.Node, state *ImportParseResult, indexInParent int, internalFuncMap walk.VisitorFuncMap[*ImportParseResult]) (*ImportParseResult, error) {
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
				return state, nil
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

			return state, nil
		}

		internalFuncMap["namespace_import"] = func(node *sitter.Node, state *ImportParseResult, indexInParent int, internalFuncMap walk.VisitorFuncMap[*ImportParseResult]) (*ImportParseResult, error) {
			state.Import = importNode

			aliasNode := importNode.NamedChild(0)
			if aliasNode == nil {
				return state, nil
			}

			alias := aliasNode.Content(content)

			imp := ImportIdentifier{
				ForeignIdentifier: "*",
				IsType:            isType,
				LocalIdentifier:   alias,
			}
			state.Imports = append(state.Imports, imp)

			return state, nil
		}

		importParseResult := ImportParseResult{IsType: isType}

		packageStringNode := node.ChildByFieldName("source")
		if packageStringNode != nil {
			packageNode := packageStringNode.NamedChild(0)
			if packageNode != nil {
				importParseResult.Package = packageNode.Content(content)
			}
		}

		_, err := walk.WalkTypeScript(node, &importParseResult, internalFuncMap)
		if err != nil {
			return state, err
		}

		return append(state, &importParseResult), nil
	}

	return walk.WalkTypeScript(node, []*ImportParseResult{}, funcMap)
}

func doExtractDynamicImports(node *sitter.Node, content []byte) ([]string, error) {
	funcMap := walk.NewVisitorFuncsMap[[]string]()
	// Do I need to do `require()` too? Require is just an `(identifier)` so there will likely be a performance hit
	funcMap["import"] = func(node *sitter.Node, state []string, indexInParent int, funcMap walk.VisitorFuncMap[[]string]) ([]string, error) {
		call := node.Parent()
		if call == nil || call.Type() != "call_expression" {
			return state, nil
		}

		arguments := call.ChildByFieldName("arguments")
		if arguments == nil || arguments.NamedChildCount() != 1 {
			return state, nil
		}

		string := arguments.NamedChild(0)
		if string == nil {
			return state, nil
		}

		fragment := string.NamedChild(0)
		if fragment == nil || fragment.Type() != "string_fragment" {
			return state, nil
		}

		return append(state, fragment.Content(content)), nil
	}

	return walk.WalkTypeScript(node, []string{}, funcMap)
}

func ExtractDynamicImports(node *sitter.Node, content []byte) ([]string, error) {
	if node != nil {
		return doExtractDynamicImports(node, content)
	}

	root, err := utils.ParseText(content, utils.TypeScript)
	if err != nil {
		return []string{}, err
	}

	return doExtractDynamicImports(root, content)
}

func ExtractImports(node *sitter.Node, content []byte) ([]*ImportParseResult, error) {
	if node != nil {
		return doExtractImports(node, content)
	}

	root, err := utils.ParseText(content, utils.TypeScript)
	if err != nil {
		return []*ImportParseResult{}, err
	}

	return doExtractImports(root, content)
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
			text = fmt.Sprintf("import type {%s} from '%s';", strings.Join(toAdd, ", "), packageName)
		} else {
			text = fmt.Sprintf("import {%s} from '%s';", strings.Join(toAdd, ", "), packageName)
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
					if !packageIsRelative && resultIsRelative && index > 0 {
						r = importResults[index-1]
					}

					startPoint := r.Import.StartPoint()
					editRange = utils.Range{Start: utils.PositionFromPoint(startPoint), End: utils.PositionFromPoint(startPoint)}
					text = text + "\n"

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
	importEdits = append(importEdits, AddToImport(importResults, packageName, toAddTypes, true)...)

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
