package analysis

import (
	"ts_inspector/parser"
)

func asyncAngular(_ *parser.State, file *parser.File) ([]Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]Analysis, error) {
		analyses := []Analysis{}

		for _, definition := range class.Snapshot().Definitions.All() {
			if definition.IsAngularesqueMethod && definition.Async {
				message := "Angular method must not be async"
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Error, "async-angular", message))
			}
		}

		return analyses, nil
	}), nil
}
