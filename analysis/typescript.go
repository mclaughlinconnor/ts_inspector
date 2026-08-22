package analysis

import (
	"slices"
	"strconv"
	"strings"
	"ts_inspector/config"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

var excludedCodes = []int32{
	2540, // Cannot assign to '' because it is a read-only property.
}

func shouldSkip(diagnostic *parser.Diagnostic) bool {
	return slices.Contains(excludedCodes, diagnostic.Code)
}

func typescript(state *parser.State, file *parser.File) ([]Analysis, error) {
	analyses := []Analysis{}

	if file.Snapshot().Filetype != "pug" {
		return analyses, nil
	}

	d := state.GetTsGo().GetSemanticDiagnostics(file.GetTcbUri())
	if d == nil {
		return analyses, nil
	}

	typescriptDiagnostics := d.Result

	tcbBlock, err := tcb.BuildTcbBlock(state, file)
	if err != nil {
		return analyses, err
	}

	for _, diagnostic := range typescriptDiagnostics {
		if shouldSkip(&diagnostic) {
			continue
		}

		r := tcbBlock.TsOffsetToRange(file.Snapshot().Content, diagnostic.Pos, diagnostic.End, config.GetConfig().Debug)
		if r == nil {
			continue
		}

		code := "typescript-" + strconv.Itoa(int(diagnostic.Code))

		relatedInformation := []RelatedInformation{}
		for _, ri := range diagnostic.RelatedInformation {
			ri := RelatedInformation{ri.Text, parser.UriFromFilename(ri.FileName), utils.ZeroRange()}
			relatedInformation = append(relatedInformation, ri)
		}

		text := strings.TrimRight(flattenText(&diagnostic, 0), "\n")
		analyses = append(analyses, newAnalysis(code, *r, AnalysisSeverityFromTsGoCategory(&diagnostic.Category), text, &relatedInformation))
	}

	return analyses, nil
}

func flattenText(diagnostic *parser.Diagnostic, depth int) string {
	sb := strings.Builder{}

	for range depth * 2 {
		sb.WriteRune(' ')
	}

	sb.WriteString(diagnostic.Text)
	sb.WriteRune('\n')

	for _, d := range diagnostic.MessageChain {
		sb.WriteString(flattenText(d, depth+1))
	}

	return sb.String()
}
