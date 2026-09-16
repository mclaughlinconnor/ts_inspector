package analysis

import (
	"fmt"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
)

func unusedAngular(_ *parser.State, file *parser.File) ([]interfaces.Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]interfaces.Analysis, error) {
		analyses := []interfaces.Analysis{}

		for _, definition := range class.Snapshot().Definitions.All() {
			if definition.HasAngularDecorator() && !definition.IsUsed() && !definition.IsLocalParam() {
				code := "unused-angular"
				if definition.Override {
					message := fmt.Sprintf("Angular property never used in this component: %s. Check the parent class.", definition.Name)
					analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, interfaces.AnalysisSeverity.Hint, code, message))
				} else {
					message := fmt.Sprintf("Angular property never used in this component: %s", definition.Name)
					analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, interfaces.AnalysisSeverity.Warning, code, message))
				}
			}
		}

		return analyses, nil
	}), nil
}
