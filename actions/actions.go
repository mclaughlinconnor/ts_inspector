package actions

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

var Actions []Action

type Action struct {
	Perform func(parser.State, parser.File, utils.Range) (actionEdits []utils.TextEdit, allowed bool, err error)
	Title   string
}

func registerAction(action Action) {
	Actions = append(Actions, action)
}

func InitActions() {
	registerAction(Action{ImplementAngularAfterViewInit, "Add AfterViewInit"})
	registerAction(Action{ImplementAngularOnChanges, "Add OnChanges"})
	registerAction(Action{ImplementAngularOnDestroy, "Add OnDestroy"})
	registerAction(Action{ImplementAngularOnInit, "Add OnInit"})
	registerAction(Action{MakeAsync, "Make surrounding method async"})
	registerAction(Action{RearrangeClass, "Rearrange class"})
}
