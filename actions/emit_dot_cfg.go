package actions

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func SaveDotForCfg(
	_ *utils.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	args := []any{file.Snapshot().URI}

	command = &interfaces.Command{
		Title:     "Save dot graph for CFG",
		Command:   "ts_inspector/saveDotCfg",
		Arguments: &args[0],
	}

	return nil, command, true, nil
}
