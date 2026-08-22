package commands

import (
	"errors"
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func AddImport(_ *utils.Writer, state *parser.State, args *any) (map[string]utils.TextEdits, error) {
	changes := map[string]utils.TextEdits{}
	slice, ok := (*args).([]any)

	if !ok {
		return changes, errors.New("the args aren't an array")
	}

	if len(slice) != 4 {
		return changes, errors.New("the slice does not contain exactly three elements")
	}

	uri, ok1 := slice[0].(string)
	packageName, ok2 := slice[1].(string)
	_typeImports, ok3 := slice[2].([]any)
	_imports, ok4 := slice[3].([]any)

	if !ok1 || !ok2 || !ok3 || !ok4 {
		return changes, errors.New("one or more elements are not strings")
	}

	typeImports := make([]string, len(_typeImports))
	for i, v := range _typeImports {
		typeImports[i], ok3 = v.(string)
		if !ok3 {
			return changes, errors.New("the fourth element contains non-string elements")
		}
	}

	imports := make([]string, len(_imports))
	for i, v := range _imports {
		imports[i], ok4 = v.(string)
		if !ok4 {
			return changes, errors.New("the fourth element contains non-string elements")
		}
	}

	file, found := state.GetFile(parser.FilenameFromUri(uri))
	if !found {
		return changes, nil
	}

	edits, err := ast.AddImportToFile([]byte(file.Snapshot().Content), packageName, imports, typeImports)
	if err != nil {
		return changes, err
	}

	changes[uri] = edits
	return changes, nil
}
