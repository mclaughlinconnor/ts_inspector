package actions

import (
	"io"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func GoToDeclaringModule(
	_ io.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	rangeStartPosition := editRange.Start
	rangeEndPosition := editRange.End

	rangeStartOffset := file.GetOffsetForPosition(rangeStartPosition)
	rangeEndOffset := file.GetOffsetForPosition(rangeEndPosition)

	args := []any{file.Snapshot().URI, rangeStartOffset, rangeEndOffset}
	anyArgs := any(args)

	command = &interfaces.Command{
		Title:     "Go to declaring module",
		Command:   "ts_inspector/goToDeclaringModule",
		Arguments: &anyArgs,
	}

	return nil, command, true, nil
}
