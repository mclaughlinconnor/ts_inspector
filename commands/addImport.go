package commands

import (
	"errors"
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func AddImport(state parser.State, args *[]any) (map[string]utils.TextEdits, error) {
	changes := map[string]utils.TextEdits{}
	slice := *args

	if len(slice) != 4 {
		return changes, errors.New("the slice does not contain exactly three elements")
	}

	uri, ok1 := slice[0].(string)
	packageName, ok2 := slice[1].(string)
	_typeImports, ok3 := slice[2].([]interface{})
	_imports, ok4 := slice[3].([]interface{})

	if !ok1 || !ok2 || !ok3 || !ok4 {
		return changes, errors.New("one or more elements are not strings")
	}

	typeImports := make([]string, len(_typeImports))
	for i, v := range _typeImports {
		typeImports[i], ok3 = v.(string)
		if !ok4 {
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

	file := state[parser.FilenameFromUri(uri)]

	return utils.ParseFile(false, file.Content, utils.TypeScript, changes, func(root *sitter.Node, content []byte, changes map[string]utils.TextEdits) (map[string]utils.TextEdits, error) {
		edits, err := ast.AddImportToFile(content, packageName, imports, typeImports)
		if err != nil {
			return changes, err
		}

		changes[uri] = edits
		return changes, nil
	})
}
