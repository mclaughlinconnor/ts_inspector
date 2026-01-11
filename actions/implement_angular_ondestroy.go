package actions

import (
	"io"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ImplementAngularOnDestroy(_ io.Writer, state *parser.State, file *parser.File, _ utils.Range) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	return ImplementAngular(
		state,
		file,
		"OnDestroy",
		[]string{"OnDestroy"},
		"  /** @inheritDoc */\n  public ngOnDestroy(): void {\n\n  }",
		"ngOnDestroy",
		2,
	)
}
