package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func constructorOnlyProperty(file *parser.File) []Analysis {
	return analyseClasses(file, func(class parser.Class) []Analysis {
		analyses := []Analysis{}

		for _, definition := range class.Definitions {
			used := len(definition.Usages) != 0

			if used && definition.IsConstructorParam() && definition.UsageAccess == parser.ConstructorAccess {
				message := fmt.Sprintf("Variable only used in constructor: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, "constructor-only-property", message))
			}
		}

		return analyses
	})
}
