package analysis

import (
	"ts_inspector/parser"
)

func angularManyDecorators(state *parser.State, file *parser.File) []Analysis {
	return analyseClasses(file, func(class *parser.Class) []Analysis {
		analyses := []Analysis{}

		if class.Snapshot().Angular != nil && class.Snapshot().Angular.Component != nil && class.Snapshot().Angular.Module != nil {
			message := "Class cannot be both a @Component and a @NgModule at the same time"
			analyses = append(analyses, newAnalysisHighlightName(class.Snapshot().NameNode, class, AnalysisSeverity.Error, "angular-method-no-many-decorators", message))
		}

		return analyses
	})
}
