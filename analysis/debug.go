package analysis

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func debug(file *parser.File) []Analysis {
	analyses := []Analysis{}

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
			if c.HasComponent() && c.Angular.Component.DeclaredIn != nil {
				hasDeclaredIn = true
			}
		}

		if !hasDeclaredIn {
			analyses = append(analyses, newAnalysis("code", utils.Range{Start: one, End: one}, 2, "Is not declared anywheere"))
		}
	}

	return analyses
}
