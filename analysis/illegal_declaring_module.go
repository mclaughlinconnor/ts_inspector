package analysis

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func illegalDeclaringModule(state *parser.State, file *parser.File) ([]interfaces.Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]interfaces.Analysis, error) {
		analyses := []interfaces.Analysis{}

		if class.Snapshot().Angular == nil || class.Snapshot().Angular.Module == nil {
			return analyses, nil
		}

		for declaration := range class.Snapshot().Angular.Module.Declarations.FlattenReferenceArraysToReferences(state) {
			declaration.Resolve(state)

			if declaration != nil && declaration.Class != nil && declaration.Class.Snapshot().Angular != nil && declaration.Class.Snapshot().Angular.Module != nil {
				n := declaration.Node

				startPosition := utils.LspPositionFromTsPosition(n.StartPosition())
				endPosition := utils.LspPositionFromTsPosition(n.EndPosition())

				r := utils.Range{Start: startPosition, End: endPosition}

				message := "Angular NgModule may not declare another NgModule"
				analyses = append(analyses, interfaces.NewAnalysis("illegal-declaring-module", r, interfaces.AnalysisSeverity.Error, message, nil))
			}
		}

		return analyses, nil
	}), nil
}
