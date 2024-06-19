package actions

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ImplementAngularOnChanges(state parser.State, file parser.File, _ utils.Range) (actionEdits utils.TextEdits, allowed bool, err error) {
	return ImplementAngular(state, file, "OnChanges", []string{"OnChanges", "SimpleChanges"}, "  /** @inheritDoc */\n  public ngOnChanges(changes: SimpleChanges): void {\n\n  }", "ngOnChanges")
}
