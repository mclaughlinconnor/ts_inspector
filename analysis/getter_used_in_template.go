package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func getterUsedInTemplate(file parser.File) []Analysis {
	return analyseClasses(file, func(class parser.Class) []Analysis {
		analyses := []Analysis{}

		if class.AngularTemplateFile == nil {
			return analyses
		}

		for _, definition := range class.GetGetters() {
			used := len(definition.Usages) != 0
			if used && definition.UsageAccess == parser.TemplateAccess {
				message := fmt.Sprintf("Getter used in template: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Hint, "getter-used-in-template", message))
			}
		}

		return analyses
	})
}
