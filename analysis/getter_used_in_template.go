package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func getterUsedInTemplate(_ *parser.State, file *parser.File) ([]Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]Analysis, error) {
		analyses := []Analysis{}

		if class.GetTemplateFile() == nil {
			return analyses, nil
		}

		for _, definition := range class.GetGetters() {
			used := len(definition.Usages) != 0
			if used && definition.UsageAccess == parser.TemplateAccess {
				message := fmt.Sprintf("Getter used in template: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Hint, "getter-used-in-template", message))
			}
		}

		return analyses, nil
	}), nil
}
