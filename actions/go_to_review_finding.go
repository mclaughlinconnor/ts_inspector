package actions

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func GoToReviewFinding(
	_ *utils.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	position := editRange.Start
	line := position.Line

	args := []any{file.Snapshot().URI, line}
	anyArgs := any(args)

	command = &interfaces.Command{
		Title:     "Go to review finding under cursor",
		Command:   "ts_inspector/gotoReviewFinding",
		Arguments: &anyArgs,
	}

	return nil, command, true, nil
}
