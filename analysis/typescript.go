package analysis

import (
	"strconv"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb_cm"
	"ts_inspector/utils"
)

func typescript(state *parser.State, file *parser.File) []Analysis {
	analyses := []Analysis{}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	d := state.GetTsGo().GetSemanticDiagnostics(file.GetTcbUri())
	typescriptDiagnostics := d.Result

	tcbBlock, _ := tcb_cm.BuildTcbBlock(state, file)

	for _, diagnostic := range typescriptDiagnostics {
		tcbBlock.TsToPugLocation(diagnostic.Pos, diagnostic.End, func(part tcb_cm.Part) {
			var start utils.Position
			var end utils.Position

			if part.IsReal() {
				start = utils.GetPositionForOffset(file.Snapshot().Content, uint32(*part.PugStartOffset))
				end = utils.GetPositionForOffset(file.Snapshot().Content, uint32(*part.PugEndOffset))
			} else if utils.Debug {
				start = utils.Position{Line: 0, Character: 0}
				end = utils.Position{Line: 0, Character: 0}
			} else {
				return
			}

			r := utils.Range{Start: start, End: end}
			code := "typescript-" + strconv.Itoa(int(diagnostic.Code))

			analyses = append(analyses, newAnalysis(code, r, AnalysisSeverityFromTsGoCategory(&diagnostic.Category), diagnostic.Text))

			for _, ri := range diagnostic.RelatedInformation {
				code := "typescript-" + strconv.Itoa(int(ri.Code))
				analyses = append(analyses, newAnalysis(code, r, AnalysisSeverityFromTsGoCategory(&ri.Category), ri.Text))
			}
		})
	}

	return analyses
}
