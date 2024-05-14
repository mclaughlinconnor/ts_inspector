package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func Analyse(file parser.File) []Analysis {
	analyses := []Analysis{}

	getters := file.GetGetters()
	usages := file.Usages
	vars := file.Definitions

	for _, definition := range getters {
		usage, used := usages[definition.Name]
		if used && usage.Access == parser.ForeignAccess {
			message := fmt.Sprintf("Getter used in template: %s", definition.Name)
			analyses = append(analyses, Analysis{definition.Node, AnalysisSeverity.Hint, "ts_inspector", message})
		}
	}

	for _, definition := range vars {
		definitionIsPublic := definition.AccessModifier == parser.PublicAccessibility
		usage, used := usages[definition.Name]

		if used && usage.Access == parser.ConstructorAccess {
			message := fmt.Sprintf("Variable only used in constructor: %s", definition.Name)
			analyses = append(analyses, Analysis{definition.Node, AnalysisSeverity.Warning, "ts_inspector", message})
			continue
		}

		if definitionIsPublic {
			if !used {
				message := fmt.Sprintf("Unused public variable: %s", definition.Name)
				analyses = append(analyses, Analysis{definition.Node, AnalysisSeverity.Warning, "ts_inspector", message})
			} else if usage.Access != parser.ForeignAccess {
				message := fmt.Sprintf("Needlessly public variable: %s", definition.Name)
				analyses = append(analyses, Analysis{definition.Node, AnalysisSeverity.Warning, "ts_inspector", message})
			}
		}
	}

	return analyses
}
