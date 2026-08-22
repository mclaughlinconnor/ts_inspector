package actions

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ViewTcbFile(
	_ *utils.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	args := []any{file.Snapshot().URI}
	anyArgs := any(args)

	command = &interfaces.Command{
		Title:     "View the TCB for the template",
		Command:   "ts_inspector/viewTcb",
		Arguments: &anyArgs,
	}

	return nil, command, true, nil
}
