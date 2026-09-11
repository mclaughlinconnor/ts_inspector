package actions

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func GetReviewFindings(
	_ *utils.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	command = &interfaces.Command{
		Title:     "Get all review findings",
		Command:   "ts_inspector/getReviewFindings",
		Arguments: nil,
	}

	return nil, command, true, nil
}
