package actions

import (
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

var actions []Action

type actionEditHolder struct {
	Edits     utils.TextEdits
	IsAllowed bool
}

type Action struct {
	Perform func(*utils.Writer, *parser.State, *parser.File, utils.Range) (actionEdits *[]utils.TextEdit, command *interfaces.Command, allowed bool, err error)
	Title   string
}

func GetActions(state *parser.State, file *parser.File, offset int) ([]Action, error) {
	allowedActions := []Action{}
	allowedActions = append(allowedActions, actions...)

	if file.Snapshot().Filetype == "typescript" {
		ast := file.Snapshot().Ast
		if ast == nil {
			return []Action{}, nil
		}

		for _, astAction := range ast.GetAllActions(uint(offset)) {
			allowedActions = append(allowedActions, Action{
				Perform: func(w *utils.Writer, s *parser.State, f *parser.File, r utils.Range) (actionEdits *[]utils.TextEdit, command *interfaces.Command, allowed bool, err error) {
					edits, err := astAction.Perform()
					return &edits, nil, true, err
				},
				Title: astAction.Name,
			})
		}

	}

	return allowedActions, nil
}

func registerAction(action Action) {
	actions = append(actions, action)
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
	registerAction(Action{GoToReviewFinding, "Go to review finding under cursor"})
	registerAction(Action{ImplementAngularAfterViewInit, "Add AfterViewInit"})
	registerAction(Action{ImplementAngularOnChanges, "Add OnChanges"})
	registerAction(Action{ImplementAngularOnDestroy, "Add OnDestroy"})
	registerAction(Action{ImplementAngularOnInit, "Add OnInit"})
	registerAction(Action{MakeAsync, "Make surrounding method async"})
	registerAction(Action{RearrangeClass, "Rearrange class"})
	registerAction(Action{ReindexReviewFindings, "Re-index review findings"})
	registerAction(Action{GetReviewFindings, "Get review findings"})
	registerAction(Action{ViewTcbFile, "View the TCB for the template"})

	if config.GetConfig().Debug {
		registerAction(Action{SaveDotForCfg, "Save dot graph for CFG"})
	}
}
