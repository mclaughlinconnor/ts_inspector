package actions

import (
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func AddDestroyedObservable(state parser.State, file parser.File, _ utils.Range) (actionEdits utils.TextEdits, allowed bool, err error) {
	if file.Filetype != "typescript" {
		return nil, false, nil
	}

	action := addDestroyedAction{[]utils.TextEdit{}, true}
	action, err = utils.ParseFile(false, file.Content, utils.TypeScript, action, func(root *sitter.Node, content []byte, edits addDestroyedAction) (addDestroyedAction, error) {
		definitionResults := ast.ExtractDefinitions(content)
		definitionResult, err := ast.FindDefinition(&definitionResults, "_destroyed$")
		if err != nil || definitionResult != nil {
			return addDestroyedAction{[]utils.TextEdit{}, false}, err
		}

		imports := [][]string{
			{"@aloreljs/rxutils", "nextComplete", ""},
			{"rxjs", "AsyncSubject", ""},
			{"@angular/core", "", "OnDestroy"},
			{"@angular/core", "", "OnInit"},
		}

		for _, i := range imports {
			pkg := i[0]
			valImport := i[1]
			typeImport := i[2]

			var valImports = []string{}
			if valImport != "" {
				valImports = []string{valImport}
			}

			var typeImports = []string{}
			if typeImport != "" {
				typeImports = []string{typeImport}
			}

			importEdits, err := ast.AddImportToFile(content, pkg, valImports, typeImports)
			if err != nil {
				return action, err
			}

			action.Edits = append(action.Edits, importEdits...)
		}

		for _, imp := range []string{"OnDestroy", "OnInit"} {
			implementEdits, err := ast.AddImplementToFile(content, imp)
			if err != nil {
				return edits, err
			} else if len(implementEdits) == 1 {
				action.Edits = append(action.Edits, implementEdits[0])
			}
		}

		propertyEdits, err := ast.AddMethodDefinitionToFile(content, "  private _destroyed$: AsyncSubject<void>;", "_destroyed$", 300)
		if err != nil {
			return edits, err
		}

		action.Edits = append(action.Edits, propertyEdits...)

		onInitEdits, err := addNgOnInit(content)
		if err != nil {
			return action, err
		}

		action.Edits = append(action.Edits, onInitEdits...)

		onDestroyEdits, err := addNgOnDestroy(content)
		if err != nil {
			return action, err
		}

		action.Edits = append(action.Edits, onDestroyEdits...)

		return action, nil
	})

	return action.Edits, action.IsAllowed, err
}

func addNgOnInit(content []byte) (utils.TextEdits, error) {
	assignment := "this._destroyed$ = new AsyncSubject();"
	edits, err := addOrPrependMethod(content, "ngOnInit", assignment)

	return edits, err
}

func addNgOnDestroy(content []byte) (utils.TextEdits, error) {
	assignment := "if (this._destroyed$) {\n      nextComplete(this._destroyed$);\n    }"
	edits, err := addOrPrependMethod(content, "ngOnDestroy", assignment)

	return edits, err
}

func addOrPrependMethod(content []byte, methodName string, toPrepend string) (utils.TextEdits, error) {
	var edits = utils.TextEdits{}

	definitionResults := ast.ExtractDefinitions(content)

	definitionResult, err := ast.FindDefinition(&definitionResults, methodName)
	if err != nil {
		return edits, err
	}

	if definitionResult == nil {
		bodyText := "  /** @inheritDoc */\n  public " + methodName + "(): void {\n    " + toPrepend + "\n  }"
		methodEdits, err := ast.AddMethodDefinitionToFile(content, bodyText, methodName, 2)
		if err != nil {
			return edits, err
		}

		edits = append(edits, methodEdits...)
	} else {
		node := definitionResult.DefinitionNode
		bodyNode := node.ChildByFieldName("body")
		if bodyNode == nil {
			return edits, nil
		}

		if bodyNode.NamedChildCount() == 0 {
			editRange := utils.Range{Start: utils.PositionFromPoint(bodyNode.StartPoint()), End: utils.PositionFromPoint(bodyNode.EndPoint())}
			text := "{\n  " + toPrepend + "}"

			edits = append(edits, utils.TextEdit{Range: editRange, NewText: text})

			return edits, nil
		}

		start := bodyNode.NamedChild(0).StartPoint()
		editRange := utils.Range{Start: utils.PositionFromPoint(start), End: utils.PositionFromPoint(start)}
		edits = append(edits, utils.TextEdit{Range: editRange, NewText: toPrepend + "\n    "})
	}

	return edits, nil
}

type addDestroyedAction struct {
	Edits     utils.TextEdits
	IsAllowed bool
}
