package analysis

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func recursiveTemplate(_ *parser.State, file *parser.File) []Analysis {
	analyses := []Analysis{}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	for _, class := range file.Snapshot().Classes {
		if class.Snapshot().Angular == nil || class.Snapshot().Angular.Component == nil || class.Snapshot().Angular.Component.Template == nil {
			continue
		}

		component := class.Snapshot().Angular.Component

		message := "Component recursively uses itself"

		for _, selector := range component.Selectors {
			// TODO: selectors aren't necessarily the tag name
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
