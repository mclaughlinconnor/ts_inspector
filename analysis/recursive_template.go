package analysis

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func recursiveTemplate(file *parser.File) []Analysis {
	analyses := []Analysis{}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	for _, class := range file.Snapshot().Classes {
		if class.Angular == nil || class.Angular.Component == nil || class.Angular.Component.Template == nil {
			continue
		}

		component := class.Angular.Component
		selector := component.Selector

		usage, found := component.Template.TagUsages[selector]
		if found && len(usage.Usages) > 0 {
			message := "Component recursively uses itself"

			for _, u := range component.Template.TagUsages[selector].Usages {
				startPosition := utils.PositionFromPoint(u.Node.StartPoint())
				endPosition := utils.PositionFromPoint(u.Node.EndPoint())

				r := utils.Range{Start: startPosition, End: endPosition}
				analyses = append(analyses, newAnalysis("angular-recursive-component", r, AnalysisSeverity.Information, message))
			}
		}

	}

	return analyses
}
