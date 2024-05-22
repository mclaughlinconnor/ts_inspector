package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func Analyse(file parser.File) []Analysis {
	analyses := []Analysis{}

	getters := file.GetGetters()
	definitions := file.Definitions

	for _, definition := range getters {
		used := len(definition.Usages) != 0
		if used && definition.UsageAccess == parser.ForeignAccess {
			message := fmt.Sprintf("Getter used in template: %s", definition.Name)
			analyses = append(analyses, Analysis{definition.Node, AnalysisSeverity.Hint, "ts_inspector", message})
		}
	}

	for _, definition := range definitions {
		definitionIsPublic := definition.AccessModifier == parser.PublicAccessibility
		used := len(definition.Usages) != 0

		if used && definition.UsageAccess == parser.ConstructorAccess {
			message := fmt.Sprintf("Variable only used in constructor: %s", definition.Name)
			analyses = append(analyses, Analysis{definition.Node, AnalysisSeverity.Warning, "ts_inspector", message})
			continue
		}

		var hasAngularDecorator bool = false
		for _, decorator := range definition.Decorators {
			hasAngularDecorator = hasAngularDecorator || decorator.IsAngular
		}

		if definitionIsPublic && !hasAngularDecorator && !definition.Static && !definition.IsAngularMethod {
			if !used {
				message := fmt.Sprintf("Unused public variable: %s", definition.Name)
				analyses = append(analyses, Analysis{definition.Node, AnalysisSeverity.Warning, "ts_inspector", message})
			} else if definition.UsageAccess != parser.ForeignAccess {
				message := fmt.Sprintf("Needlessly public variable: %s", definition.Name)
				analyses = append(analyses, Analysis{definition.Node, AnalysisSeverity.Warning, "ts_inspector", message})
			}
		}
	}

	return analyses
}
