package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func constructorOnlyProperty(_ *parser.State, file *parser.File) ([]Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]Analysis, error) {
		analyses := []Analysis{}

		for _, definition := range class.Snapshot().Definitions.All() {
			used := len(definition.Usages) != 0

			if used && definition.IsConstructorParam() && definition.UsageAccess == parser.ConstructorAccess {
				message := fmt.Sprintf("Variable only used in constructor: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, "constructor-only-property", message))
			}
		}

		return analyses, nil
	}), nil
}
