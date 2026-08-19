package actions

import (
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ImplementAngular(
	state *parser.State,
	file *parser.File,
	implements string,
	imports []string,
	methodDefinition string,
	methodName string,
	score int,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	if file.Snapshot().Filetype != "typescript" {
		return nil, nil, false, nil
	}

	content := []byte(file.Snapshot().Content)

	var edits = utils.TextEdits{}

	implementEdits, err := ast.AddImplementToFile(content, implements)
	if err != nil {
		return retEdits(&edits, err)
	} else if len(implementEdits) == 1 {
		edits = append(edits, implementEdits[0])
	}

	importEdits, err := ast.AddImportToFile(content, "@angular/core", []string{}, imports)
	if err != nil {
		return retEdits(&edits, err)
	} else if len(importEdits) == 1 {
		edits = append(edits, importEdits[0])
	}

	methodEdits, err := ast.AddMethodDefinitionToFile(content, methodDefinition, methodName, score)
	if err != nil {
		return retEdits(&edits, err)
	} else if len(methodEdits) == 1 {
		edits = append(edits, methodEdits[0])
	}

	return retEdits(&edits, nil)
}
