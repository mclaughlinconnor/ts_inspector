package analysis

import (
	"ts_inspector/parser"
)

func asyncAngular(file parser.File) []Analysis {
	return analyseClasses(file, func(class parser.Class) []Analysis {
		analyses := []Analysis{}

		for _, definition := range class.Definitions {
			if definition.IsAngularesqueMethod && definition.Async {
				message := "Angular method must not be async"
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Error, "async-angular", message))
			}
		}

		return analyses
	})
}
