package actions

import (
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

var onInitMethod = "  public ngOnInit() {\n\n  }"

func ImplementAngularOnInit(state parser.State, file parser.File) (actionEdits utils.TextEdits, allowed bool, err error) {
	if file.Filetype != "typescript" {
		return nil, false, nil
	}

	var edits = utils.TextEdits{}

	action, err := utils.ParseFile(false, file.Content, utils.TypeScript, edits, func(root *sitter.Node, content []byte, edits utils.TextEdits) (utils.TextEdits, error) {
		implementEdits, err := ast.AddImplementToFile(content, "OnInit")
		if err != nil {
			return edits, err
		} else if len(implementEdits) == 1 {
			edits = append(edits, implementEdits[0])
		}

		importEdits, err := ast.AddImportToFile(content, "@angular/core", "OnInit")
		if err != nil {
			return edits, err
		} else if len(importEdits) == 1 {
			edits = append(edits, importEdits[0])
		}

		methodEdits, err := ast.AddMethodDefinitionToFile(content, onInitMethod)
		if err != nil {
			return edits, err
		} else if len(methodEdits) == 1 {
			edits = append(edits, methodEdits[0])
		}

		return edits, nil
	})

	return action, true, err
}
