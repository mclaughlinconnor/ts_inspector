package analysis

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
)

func asyncAngular(_ *parser.State, file *parser.File) ([]interfaces.Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]interfaces.Analysis, error) {
		analyses := []interfaces.Analysis{}

		for _, definition := range class.Snapshot().Definitions.All() {
			if definition.IsAngularesqueMethod && definition.Async {
				message := "Angular method must not be async"
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, interfaces.AnalysisSeverity.Error, "async-angular", message))
			}
		}

		return analyses, nil
	}), nil
}
