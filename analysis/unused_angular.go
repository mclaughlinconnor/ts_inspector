package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func unusedAngular(file *parser.File) []Analysis {
	return analyseClasses(file, func(class parser.Class) []Analysis {
		analyses := []Analysis{}

		for _, definition := range class.Definitions {
			if definition.HasAngularDecorator() && !definition.IsUsed() && !definition.IsLocalParam() {
				code := "unused-angular"
				if definition.Override {
					message := fmt.Sprintf("Angular property never used in this component: %s. Check the parent class.", definition.Name)
					analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Hint, code, message))
				} else {
					message := fmt.Sprintf("Angular property never used in this component: %s", definition.Name)
					analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, code, message))
				}
			}
		}

		return analyses
	})
}
