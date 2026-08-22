package actions

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ImplementAngularAfterViewInit(_ *utils.Writer, state *parser.State, file *parser.File, _ utils.Range) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	return ImplementAngular(
		state,
		file,
		"AfterViewInit",
		[]string{"AfterViewInit"},
		"  /** @inheritDoc */\n  public ngAfterViewInit(): void {\n\n  }",
		"ngAfterViewInit",
		2,
	)
}
