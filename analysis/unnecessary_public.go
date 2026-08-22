package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func unnecessaryPublic(_ *parser.State, file *parser.File) ([]Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]Analysis, error) {
		analyses := []Analysis{}

		for _, definition := range class.Snapshot().Definitions.All() {
			if definition.IsPublic() && !definition.HasAngularDecorator() && !definition.Static && !definition.IsAngularesqueMethod && !definition.Override {
				code := "unnecessary-public"
				if !definition.IsUsed() {
					message := fmt.Sprintf("Unused public variable: %s", definition.Name)
					analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, code, message))
				} else if definition.UsageAccess != parser.TemplateAccess {
					message := fmt.Sprintf("Needlessly public variable: %s", definition.Name)
					analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, code, message))
				}
			}
		}

		return analyses, nil
	}), nil
}
