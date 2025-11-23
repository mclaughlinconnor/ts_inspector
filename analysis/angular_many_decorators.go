package analysis

import (
	"ts_inspector/parser"
)

func angularManyDecorators(file *parser.File) []Analysis {
	return analyseClasses(file, func(class parser.Class) []Analysis {
		analyses := []Analysis{}

		if class.Angular != nil && class.Angular.Component != nil && class.Angular.Module != nil {
			message := "Class cannot be both a @Component and a @NgModule at the same time"
			analyses = append(analyses, newAnalysisHighlightName(class.NameNode, class, AnalysisSeverity.Error, "angular-method-no-many-decorators", message))
		}

		return analyses
	})
}
