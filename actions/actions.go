package actions

import (
	"io"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

var Actions []Action

type Action struct {
	Perform func(io.Writer, *parser.State, *parser.File, utils.Range) (actionEdits *[]utils.TextEdit, command *interfaces.Command, allowed bool, err error)
	Title   string
}

func registerAction(action Action) {
	Actions = append(Actions, action)
}

func InitActions() {
	registerAction(Action{AddDestroyedObservable, "Add _destroyed$ observable"})
	registerAction(Action{CalculateAllProviders, "Calculate all providers"})
	registerAction(Action{ConvertInjectToProperty, "Convert constructor injection to inject() property"})
	registerAction(Action{GoToDeclaringModule, "Go to declaring module"})
	registerAction(Action{ViewTcbFile, "View the TCB for the template"})
	registerAction(Action{ImplementAngularAfterViewInit, "Add AfterViewInit"})
	registerAction(Action{ImplementAngularOnChanges, "Add OnChanges"})
	registerAction(Action{ImplementAngularOnDestroy, "Add OnDestroy"})
	registerAction(Action{ImplementAngularOnInit, "Add OnInit"})
	registerAction(Action{MakeAsync, "Make surrounding method async"})
	registerAction(Action{RearrangeClass, "Rearrange class"})

	if utils.Debug {
		registerAction(Action{SaveDotForCfg, "Save dot graph for CFG"})
	}
}
