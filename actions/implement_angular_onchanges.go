package actions

import (
	"io"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ImplementAngularOnChanges(_ io.Writer, state *parser.State, file *parser.File, _ utils.Range) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	return ImplementAngular(
		state,
		file,
		"OnChanges",
		[]string{"OnChanges", "SimpleChanges"},
		"  /** @inheritDoc */\n  public ngOnChanges(changes: SimpleChanges): void {\n\n  }",
		"ngOnChanges",
		2,
	)
}
