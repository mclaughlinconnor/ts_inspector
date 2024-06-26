package actions

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ImplementAngularOnDestroy(state parser.State, file parser.File, _ utils.Range) (actionEdits utils.TextEdits, allowed bool, err error) {
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
