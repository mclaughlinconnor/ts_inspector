package analysis

import (
	"ts_inspector/parser"
)

func angularMethodNoImplements(_ *parser.State, file *parser.File) ([]Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]Analysis, error) {
		analyses := []Analysis{}

	OUTER:
		for _, definition := range class.Snapshot().Definitions.All() {
			if !definition.IsAngularMethod() {
				continue
			}
			for implements := range class.Snapshot().Implements.IterateResolved {
				if implements.Class.GetDefinition(definition.Name) != nil {
					continue OUTER
				}
			}

			message := "Angular method declared on class without relevant interface implementation clause"
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Error, "angular-method-no-implements", message))
		}

		return analyses, nil
	}), nil
}
