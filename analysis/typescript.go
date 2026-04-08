package analysis

import (
	"strconv"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb_cm"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func typescript(state *parser.State, file *parser.File) []Analysis {
	analyses := []Analysis{}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	d := state.GetTsGo().GetSemanticDiagnostics(file.GetTcbUri())
	typescriptDiagnostics := d.Result

	content := []byte(file.Snapshot().Content)
	tcbBlock, _ := utils.ParseText(content, utils.Pug, nil, func(root *sitter.Node, _ []byte, _ *tcb_cm.StatementParts) (*tcb_cm.StatementParts, error) {
		tcb := tcb_cm.GenerateTcb(state, file.Snapshot().Classes[0], root, content)

		return tcb, nil
	})

	for _, diagnostic := range typescriptDiagnostics {
		for _, part := range tcbBlock.Parts {
			if *part.TsStartOffset <= diagnostic.Pos && *part.TsEndOffset > diagnostic.Pos {
				var start utils.Position
				var end utils.Position

				if part.IsReal() {
					start = utils.GetPositionForOffset(file.Snapshot().Content, uint32(*part.PugStartOffset))
					end = utils.GetPositionForOffset(file.Snapshot().Content, uint32(*part.PugEndOffset))
				} else if utils.Debug {
					start = utils.Position{Line: 0, Character: 0}
					end = utils.Position{Line: 0, Character: 0}
				} else {
					continue
				}

				r := utils.Range{Start: start, End: end}
				code := "typescript-" + strconv.Itoa(int(diagnostic.Code))

				analyses = append(analyses, newAnalysis(code, r, AnalysisSeverityFromTsGoCategory(&diagnostic.Category), diagnostic.Text))
			}
		}
	}

	return analyses
}
