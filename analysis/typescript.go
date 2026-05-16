package analysis

import (
	"strconv"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

func typescript(state *parser.State, file *parser.File) []Analysis {
	analyses := []Analysis{}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	d := state.GetTsGo().GetSemanticDiagnostics(file.GetTcbUri())
	typescriptDiagnostics := d.Result

	tcbBlock, _ := tcb.BuildTcbBlock(state, file)

	for _, diagnostic := range typescriptDiagnostics {
		part := tcbBlock.TsToPugLocation(diagnostic.Pos, diagnostic.End)
		if part == nil {
			continue
		}

		var start utils.Position
		var end utils.Position

		if part.IsReal() {
			start = utils.GetPositionForOffset(file.Snapshot().Content, uint32(*part.PugStartOffset))
			end = utils.GetPositionForOffset(file.Snapshot().Content, uint32(*part.PugEndOffset))
		} else if utils.Debug {
			start = utils.ZeroPosition()
			end = utils.ZeroPosition()
		} else {
			continue
		}

		r := utils.Range{Start: start, End: end}
		code := "typescript-" + strconv.Itoa(int(diagnostic.Code))

		relatedInformation := []RelatedInformation{}
		for _, ri := range diagnostic.RelatedInformation {
			ri := RelatedInformation{ri.Text, parser.UriFromFilename(ri.FileName), utils.ZeroRange()}
			relatedInformation = append(relatedInformation, ri)
		}

		analyses = append(analyses, newAnalysis(code, r, AnalysisSeverityFromTsGoCategory(&diagnostic.Category), diagnostic.Text, &relatedInformation))
	}

	return analyses
}
