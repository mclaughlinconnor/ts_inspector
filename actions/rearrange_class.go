package actions

import (
	"cmp"
	"slices"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func RearrangeClass(
	state parser.State,
	file parser.File,
	editRange utils.Range,
) (actionEdits utils.TextEdits, allowed bool, err error) {
	definitions := ast.ExtractDefinitions([]byte(file.Content))

	if len(definitions) < 2 {
		return utils.TextEdits{}, true, nil
	}

	var sortFunc = func(a ast.MethodDefinitionParseResult, b ast.MethodDefinitionParseResult) int {
		return cmp.Or(
			cmp.Compare(b.Score, a.Score),
			cmp.Compare(a.Name, b.Name),
		)
	}

	alreadySorted := slices.IsSortedFunc(definitions, sortFunc)
	if alreadySorted {
		return utils.TextEdits{}, false, nil
	}

	start := definitions[0].Range.Start
	end := definitions[len(definitions)-1].Range.End

	slices.SortFunc(definitions, sortFunc)

	edit := utils.TextEdit{}
	edit.Range.Start = start
	edit.Range.End = end

	bodies := []string{}

	for _, definition := range definitions {
		s := file.GetOffsetForPosition(definition.Range.Start)
		e := file.GetOffsetForPosition(definition.Range.End)

		text := file.Content[s:e]
		bodies = append(bodies, text)
	}

	edit.NewText = strings.Join(bodies, "\n\n  ")

	return utils.TextEdits{edit}, true, nil
}
