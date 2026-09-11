package commands

import (
	"ts_inspector/analysis"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ReindexReviewFindings(writer *utils.Writer, state *parser.State, args *any) (map[string]utils.TextEdits, error) {
	err := state.ReindexReviewFindings()
	if err != nil {
		return map[string]utils.TextEdits{}, err
	}

	for _, file := range state.GetFiles() {
		if !file.Snapshot().IsOpen {
			continue
		}

		utils.WriteResponse(writer, analysis.GenerateDiagnosticsForFile(state, file, false))
	}

	return map[string]utils.TextEdits{}, nil
}
