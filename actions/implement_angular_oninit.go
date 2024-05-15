package actions

import (
	"fmt"
	"slices"
	"strings"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ImplementAngularOnInit(state parser.State, file parser.File) (Edits, error) {
	var edits = Edits{}

	return utils.ParseFile(false, file.Content, utils.TypeScript, edits, func(root *sitter.Node, content []byte, edits Edits) (Edits, error) {
		implementResult, _ := utils.WithMatches(utils.QueryClassImplements, utils.TypeScript, content, implementParseResult{[]string{}, nil}, func(captures []sitter.QueryCapture, returnValue implementParseResult) (implementParseResult, error) {
			for _, capture := range captures {
				if capture.Node.Type() == "implements_clause" {
					returnValue.Clause = capture.Node
				} else if capture.Node.Type() == "type_identifier" {
					returnValue.Implements = append(returnValue.Implements, capture.Node.Content(content))
				}
			}

			return returnValue, nil
		})

		if !slices.Contains(implementResult.Implements, "OnInit") {
			implementResult.Implements = append(implementResult.Implements, "OnInit")
			slices.Sort(implementResult.Implements)
			text := strings.Join(implementResult.Implements, ", ")

			node := implementResult.Clause

			editRange := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}
			editRange.Start.Character = editRange.Start.Character + uint32(len("implements "))

			edits = append(edits, Edit{editRange, text})
		}

		importResult, _ := utils.WithMatches(utils.QueryAngularImport, utils.TypeScript, content, importParseResult{[]string{}, nil}, func(captures []sitter.QueryCapture, returnValue importParseResult) (importParseResult, error) {
			for _, capture := range captures {
				fmt.Println(capture.Node.Type())
				if capture.Node.Type() == "import_clause" {
					returnValue.Clause = capture.Node
				} else if capture.Node.Type() == "identifier" {
					returnValue.Imports = append(returnValue.Imports, capture.Node.Content(content))
				}
			}

			return returnValue, nil
		})

		if !slices.Contains(importResult.Imports, "OnInit") {
			importResult.Imports = append(importResult.Imports, "OnInit")
			slices.Sort(implementResult.Implements)
			text := "{" + strings.Join(importResult.Imports, ", ") + "}"

			node := importResult.Clause
			editRange := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}

			edits = append(edits, Edit{editRange, text})
		}

		return edits, nil
	})
}

type implementParseResult struct {
	Implements []string
	Clause     *sitter.Node
}

type importParseResult struct {
	Imports []string
	Clause  *sitter.Node
}
