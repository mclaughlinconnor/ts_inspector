package analysis

import (
	"fmt"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func debug(file *parser.File) []Analysis {
	analyses := []Analysis{}

	analyses = append(analyses, analyseClasses(file, func(class *parser.Class) []Analysis {
		analyses := []Analysis{}

		for _, definition := range class.Snapshot().Definitions {
			message := fmt.Sprintf("Usages: %d", len(definition.Usages))
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Hint, "", message))
		}

		return analyses
	})...)

	for _, variable := range file.Snapshot().Variables {
		node := variable.Node

		startByte := node.StartByte()
		endByte := node.EndByte()

		content := file.Snapshot().Content

		startPosition := parser.GetPositionForOffset(content, startByte)
		endPosition := parser.GetPositionForOffset(content, endByte)

		message := "Found " + variable.Kind + " `" + variable.Name + "` with value `" + variable.Value + "`"

		analyses = append(analyses, newAnalysis("variable", utils.Range{Start: startPosition, End: endPosition}, AnalysisSeverity.Warning, message))
	}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	zero := utils.Position{Line: 0, Character: 0}
	one := utils.Position{Line: 1, Character: 0}

	classes := file.Snapshot().Classes

	if len(classes) == 0 {
		analyses = append(analyses, newAnalysis("code", utils.Range{Start: zero, End: zero}, 2, "Has no classes"))
	} else {
		hasDeclaredIn := false
		for _, c := range classes {
			if c.HasComponent() && c.Snapshot().Angular.Component.DeclaredIn != nil {
				hasDeclaredIn = true
			}
		}

		if !hasDeclaredIn {
			analyses = append(analyses, newAnalysis("code", utils.Range{Start: one, End: one}, 2, "Is not declared anywheere"))
		}
	}

	return analyses
}
