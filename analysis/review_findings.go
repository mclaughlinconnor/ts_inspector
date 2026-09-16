package analysis

import (
	"fmt"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func reviewFindings(state *parser.State, file *parser.File) ([]interfaces.Analysis, error) {
	findings := state.GetReviewFindings()
	if len(findings) == 0 {
		return []interfaces.Analysis{}, nil
	}

	analyses := []interfaces.Analysis{}

	for _, finding := range findings {
		metadata := finding.Metadata
		if finding.Metadata.FilePath != file.Filename() {
			continue
		}

		content := finding.Content

		mkAnalysis := func(r utils.Range) {
			message := fmt.Sprintf("[%v]: %v", metadata.Category, content.Summary)
			analyses = append(analyses, interfaces.NewAnalysis(finding.Content.Agent, r, interfaces.AnalysisSeverity.Error, message, nil))
		}

		if finding.Metadata.NewLine != 0 {
			line := finding.Metadata.NewLine - 1
			r := getStartAndEndOffsetFromLine(file, line)
			mkAnalysis(r)
		}

		if finding.Metadata.OldLine != 0 {
			line := finding.Metadata.OldLine - 1
			r := getStartAndEndOffsetFromLine(file, line)
			mkAnalysis(r)
		}
	}

	return analyses, nil
}

func getStartAndEndOffsetFromLine(file *parser.File, line int) utils.Range {
	start := file.GetOffsetForLineNumber(line)
	end := file.GetOffsetForLineNumber(line+1) - 1

	return file.GetLocationForOffset(int(start), int(end)).Range
}
