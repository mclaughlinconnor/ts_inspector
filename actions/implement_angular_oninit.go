package actions

import (
	"fmt"
	"slices"
	"strings"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ImplementAngularOnInit(state parser.State, file parser.File) (actionEdits utils.TextEdits, allowed bool, err error) {
	if file.Filetype != "typescript" {
		return nil, false, nil
	}

	var edits = utils.TextEdits{}

	action, err := utils.ParseFile(false, file.Content, utils.TypeScript, edits, func(root *sitter.Node, content []byte, edits utils.TextEdits) (utils.TextEdits, error) {
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

		if implementResult.Clause == nil {
			// TODO: add handling for classes without an implementation clause
			return edits, fmt.Errorf("Could not find implementation clause")
		}

		if !slices.Contains(implementResult.Implements, "OnInit") {
			implementResult.Implements = append(implementResult.Implements, "OnInit")
			slices.Sort(implementResult.Implements)
			text := strings.Join(implementResult.Implements, ", ")

			node := implementResult.Clause

			editRange := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}
			editRange.Start.Character = editRange.Start.Character + uint32(len("implements "))

			edits = append(edits, utils.TextEdit{Range: editRange, NewText: text})
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

		if importResult.Clause == nil {
			// TODO: add handling for classes without an implementation clause
			return edits, fmt.Errorf("Could not find import clause")
		}

		if !slices.Contains(importResult.Imports, "OnInit") {
			importResult.Imports = append(importResult.Imports, "OnInit")
			slices.Sort(importResult.Imports)
			text := "{" + strings.Join(importResult.Imports, ", ") + "}"

			node := importResult.Clause
			editRange := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}

			edits = append(edits, utils.TextEdit{Range: editRange, NewText: text})
		}

		methodResult, _ := utils.WithMatches(utils.QueryClassProperties, utils.TypeScript, content, []propertyParseResult{}, func(captures []sitter.QueryCapture, returnValue []propertyParseResult) ([]propertyParseResult, error) {
			result := propertyParseResult{}
			for _, capture := range captures {
				node := capture.Node
				fmt.Println(node.Type())
				if node.Type() == "property_identifier" || node.Type() == "private_property_identifier" {
					result.Name = node.Content(content)
				} else if node.Type() == ";" {
					result.Node = node
					// Type will be set when the outside @node is encountered
				} else {
					result.Type = node.Type()
					result.Node = node
				}
			}

			return append(returnValue, result), nil
		})

		slices.SortFunc(methodResult, func(a propertyParseResult, b propertyParseResult) int {
			return int(a.Node.StartByte()) - int(b.Node.StartByte())
		})

		insertionIndex := -1
		for index, result := range methodResult {
			if result.Type != "public_field_definition" {
				if index == len(methodResult)-1 {
					insertionIndex = len(methodResult) - 1
				} else {
					insertionIndex = index
				}
			}
		}

		if insertionIndex != -1 {
			insertPosition := utils.PositionFromPoint(methodResult[insertionIndex].Node.StartPoint())
			insertPosition.Character = 0
			editRange := utils.Range{Start: insertPosition, End: insertPosition}

			insertionText := `  public ngOnInit() {

  }

`

			edits = append(edits, utils.TextEdit{Range: editRange, NewText: insertionText})
		}

		if insertionIndex == -1 {
			classBodyNode, err := utils.WithMatches(utils.QueryClassBody, utils.TypeScript, content, nil, func(captures []sitter.QueryCapture, returnValue *sitter.Node) (*sitter.Node, error) {
				if len(captures) == 1 && captures[0].Node != nil {
					return captures[0].Node, nil
				}

				return nil, fmt.Errorf("Could not find class body")
			})

			if err != nil {
				return edits, err
			}

			insertionText := `{
  public ngOnInit() {

  }
}`
			editRange := utils.Range{Start: utils.PositionFromPoint(classBodyNode.StartPoint()), End: utils.PositionFromPoint(classBodyNode.EndPoint())}
			edits = append(edits, utils.TextEdit{Range: editRange, NewText: insertionText})
		}

		return edits, nil
	})

	return action, true, err
}

type implementParseResult struct {
	Implements []string
	Clause     *sitter.Node
}

type importParseResult struct {
	Imports []string
	Clause  *sitter.Node
}

type propertyParseResult struct {
	Name string
	Node *sitter.Node
	Type string
}
