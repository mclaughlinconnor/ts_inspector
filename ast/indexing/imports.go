package indexing

import (
	"path/filepath"
	"strings"
	walktypescript "ts_inspector/ast/walk_typescript"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func extractImportsFromFile(filename string) ([]string, error) {
	funcMap := walktypescript.NewVisitorFuncsMap[[]string]()

	return utils.ParseFile(true, filename, utils.TypeScript, []string{}, func(root *sitter.Node, content []byte, state []string) ([]string, error) {
		funcMap["import_statement"] = func(node *sitter.Node, state []string, indexInParent int) []string {
			source := node.ChildByFieldName("source")
			if source == nil {
				return state
			}

			path := source.NamedChild(0)
			if path == nil {
				return state
			}

			// TODO: only supports relative imports
			pathString := path.Content(content)
			if !strings.HasPrefix(pathString, ".") {
				return state
			}

			resolvedFilename, found := determineFilename(filepath.Join(filepath.Dir(filename), pathString))
			if found {
				state = append(state, resolvedFilename)
			}

			return state
		}

		state = walktypescript.Walk(root, state, funcMap)

		return state, nil
	})
}

func recursivelyRetrieveImports(filename string, depth int, maxDepth int) ([]string, error) {
	state := []string{}

	imports, err := extractImportsFromFile(filename)
	if err != nil {
		return state, err
	}

	state = append(state, imports...)

	if depth > maxDepth {
		return state, nil
	}

	for _, filename := range imports {
		imports, err := recursivelyRetrieveImports(filename, depth+1, maxDepth)
		if err != nil {
			return state, err
		}

		state = append(state, imports...)
	}

	return state, nil
}

func determineFilename(baseFilename string) (string, bool) {
	if utils.FileExists(baseFilename) {
		return baseFilename, true
	}

	typescript := baseFilename + ".ts"
	if utils.FileExists(typescript) {
		return typescript, true
	}

	javascript := baseFilename + ".js"
	if utils.FileExists(javascript) {
		return javascript, true
	}

	return "", false
}
