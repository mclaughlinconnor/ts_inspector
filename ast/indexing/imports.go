package indexing

import (
	"path/filepath"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func DetermineFilename(baseFilename string) (string, bool) {
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

func extractImportsFromFile(filename string) ([]string, error) {
	root, content, err := utils.ParseTextFromPath(filename, utils.TypeScript)
	if err != nil {
		return []string{}, err
	}

	funcMap := walk.NewVisitorFuncsMap[[]string]()
	funcMap["import_statement"] = func(node *sitter.Node, state []string, indexInParent int, _ walk.VisitorFuncMap[[]string]) ([]string, error) {
		source := node.ChildByFieldName("source")
		if source == nil {
			return state, nil
		}

		path := source.NamedChild(0)
		if path == nil {
			return state, nil
		}

		// TODO: only supports relative imports
		pathString := path.Content(content)
		if !strings.HasPrefix(pathString, ".") {
			return state, nil
		}

		resolvedFilename, found := DetermineFilename(filepath.Join(utils.PathDir(filename), pathString))
		if found {
			state = append(state, resolvedFilename)
		}

		return state, nil
	}

	return walk.WalkTypeScript(root, []string{}, funcMap)
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
