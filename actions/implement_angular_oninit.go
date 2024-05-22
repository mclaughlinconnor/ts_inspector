package actions

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ImplementAngularOnInit(state parser.State, file parser.File) (actionEdits utils.TextEdits, allowed bool, err error) {
	return ImplementAngular(state, file, "OnInit", []string{"OnInit"}, "  /** @inheritDoc */\n  public ngOnInit() {\n\n  }", "ngOnInit")
}
