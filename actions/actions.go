package actions

import (
	"io"
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

var Actions []Action

type actionEditHolder struct {
	Edits     utils.TextEdits
	IsAllowed bool
}

type Action struct {
	Perform func(io.Writer, *parser.State, *parser.File, utils.Range) (actionEdits *[]utils.TextEdit, command *interfaces.Command, allowed bool, err error)
	Title   string
}

func registerAction(action Action) {
	Actions = append(Actions, action)
}

func retAction(action actionEditHolder, err error) (*utils.TextEdits, *interfaces.Command, bool, error) {
	return &action.Edits, nil, action.IsAllowed, nil
}

func retActionErr(err error) (*utils.TextEdits, *interfaces.Command, bool, error) {
	return retAction(actionEditHolder{[]utils.TextEdit{}, false}, err)
}

func retEdits(edits *utils.TextEdits, err error) (*utils.TextEdits, *interfaces.Command, bool, error) {
	return edits, nil, edits == nil || len(*edits) != 0, err
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

	if config.Debug {
		registerAction(Action{SaveDotForCfg, "Save dot graph for CFG"})
	}
}
