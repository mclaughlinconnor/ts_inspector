package analysis

import (
	"fmt"
	"ts_inspector/analysis/cfg"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func debug(_ *parser.State, file *parser.File) ([]Analysis, error) {
	analyses := []Analysis{}

	analyses = append(analyses, analyseClasses(file, func(class *parser.Class) ([]Analysis, error) {
		analyses := []Analysis{}

		for _, definition := range class.Snapshot().Definitions.All() {
			message := fmt.Sprintf("Usages: %d", len(definition.Usages))
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Hint, "", message))
		}

		return analyses, nil
	})...)

	for _, variable := range file.Snapshot().Variables {
		node := variable.Node

		startByte := node.StartByte()
		endByte := node.EndByte()

		content := file.Snapshot().Content

		startPosition := utils.GetPositionForOffset(content, startByte)
		endPosition := utils.GetPositionForOffset(content, endByte)

		value := variable.Value
		var str = ""
		if value == nil {
			str = "nil"
		} else if value.Type == "string" {
			str = value.StringValue
		} else if value.Type == "array" {
			str = "array"
		} else if value.Type == "reference" {
			str = value.Reference.Name
		} else if value.Type == "spread" {
			str = value.SpreadReference.Name
		}

		message := "Found " + variable.Kind + " `" + variable.Name + "` with value `" + str + "`"

		analyses = append(analyses, newAnalysis("variable", utils.Range{Start: startPosition, End: endPosition}, AnalysisSeverity.Warning, message, nil))
	}

	if file.Snapshot().Filetype == "typescript" {
		cfgState, err := cfg.BuildGraphFromFile(file)
		if err != nil {
			return analyses, err
		}

		for _, cfg := range cfgState.AllCfg {
			if cfg.Type == "program" {
				continue
			}

			message := fmt.Sprintf("Complexity: %v (%v edges, %v nodes)", cfg.CalculateCyclomaticComplexity(), cfg.CountDownwardEdges(), cfg.CountDownwardNodes())

			analysis := newAnalysisFromFileNode(file, "complexity", cfg.Node, AnalysisSeverity.Warning, message, nil)
			analysis.Range.End = analysis.Range.Start
			analyses = append(analyses, analysis)
		}
	}

	if file.Snapshot().Filetype != "pug" {
		return analyses, nil
	}

	zero := utils.Position{Line: 0, Character: 0}
	one := utils.Position{Line: 1, Character: 0}

	classes := file.Snapshot().Classes

	if len(classes) == 0 {
		analyses = append(analyses, newAnalysis("code", utils.Range{Start: zero, End: zero}, 2, "Has no classes", nil))
	} else {
		hasDeclaredIn := false
		for _, c := range classes {
			if c.HasComponent() && c.Snapshot().Angular.Component.DeclaredIn != nil {
				hasDeclaredIn = true
			}
		}

		if !hasDeclaredIn {
			analyses = append(analyses, newAnalysis("code", utils.Range{Start: one, End: one}, 2, "Is not declared anywhere", nil))
		}
	}

	return analyses, nil
}
