package actions

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ImplementAngularOnChanges(state parser.State, file parser.File) (actionEdits utils.TextEdits, allowed bool, err error) {
	return ImplementAngular(state, file, "OnChanges", []string{"OnChanges", "SimpleChanges"}, "  /** @inheritDoc */\n  public ngOnChanges(changes: SimpleChanges) {\n\n  }", "ngOnChanges")
}
