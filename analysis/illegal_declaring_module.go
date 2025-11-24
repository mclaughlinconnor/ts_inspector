package analysis

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func illegalDeclaringModule(file *parser.File) []Analysis {
	return analyseClasses(file, func(class *parser.Class) []Analysis {
		analyses := []Analysis{}

		if class.Snapshot().Angular == nil || class.Snapshot().Angular.Module == nil {
			return analyses
		}

		for i, declaration := range class.Snapshot().Angular.Module.Declarations {
			if declaration != nil && declaration.Class != nil && declaration.Class.Snapshot().Angular != nil && declaration.Class.Snapshot().Angular.Module != nil {
				n := class.Snapshot().Angular.Module.DeclarationsIdentNodes[i]

				startPosition := utils.PositionFromPoint(n.StartPoint())
				endPosition := utils.PositionFromPoint(n.EndPoint())

				r := utils.Range{Start: startPosition, End: endPosition}

				message := "Angular NgModule may not declare another NgModule"
				analyses = append(analyses, newAnalysis("illegal-declaring-module", r, AnalysisSeverity.Error, message))
			}
		}

		return analyses
	})
}
