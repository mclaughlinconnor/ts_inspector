package actions

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ReindexReviewFindings(
	_ *utils.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	command = &interfaces.Command{
		Title:     "Re-index review findings",
		Command:   "ts_inspector/reindexReviewFindings",
		Arguments: nil,
	}

	return nil, command, true, nil
}
