package analysis

import (
	"ts_inspector/parser"
)

func angularMethodNoImplements(file *parser.File) []Analysis {
	return analyseClasses(file, func(class parser.Class) []Analysis {
		analyses := []Analysis{}

	OUTER:
		for _, definition := range class.Definitions {
			if !definition.IsAngularMethod() {
				continue
			}
			for implements := range class.Implements.IterateResolved {
				if implements.Class.HasDefinition(definition.Name) {
					continue OUTER
				}
			}

			message := "Angular method declared on class without relevant interface implementation clause"
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Error, "angular-method-no-implements", message))
		}

		return analyses
	})
}
