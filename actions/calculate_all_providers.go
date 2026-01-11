package actions

import (
	"io"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func CalculateAllProviders(
	writer io.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	args := []any{file.Snapshot().URI}

	command = &interfaces.Command{
		Title:     "Print providers",
		Command:   "ts_inspector/printProviders",
		Arguments: &args[0],
	}

	return nil, command, true, nil
}
