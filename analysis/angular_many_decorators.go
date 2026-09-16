package analysis

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
)

func angularManyDecorators(state *parser.State, file *parser.File) ([]interfaces.Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]interfaces.Analysis, error) {
		analyses := []interfaces.Analysis{}

		if class.Snapshot().Angular != nil && class.Snapshot().Angular.Component != nil && class.Snapshot().Angular.Module != nil {
			message := "Class cannot be both a @Component and a @NgModule at the same time"
			analyses = append(analyses, newAnalysisHighlightName(class.Snapshot().NameNode, class, interfaces.AnalysisSeverity.Error, "angular-method-no-many-decorators", message))
		}

		return analyses, nil
	}), nil
}
