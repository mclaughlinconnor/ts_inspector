package analysis

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func NewDiagnosticNotification(uri string, version int, diagnostics []interfaces.Diagnostic) interfaces.PublishDiagnosticsNotification {
	return interfaces.PublishDiagnosticsNotification{
		RPC:    "2.0",
		Method: "textDocument/publishDiagnostics",
		Params: interfaces.PublishDiagnosticsParams{Uri: uri, Version: &version, Diagnostics: diagnostics},
	}
}

func GenerateDiagnosticsForFile(state *parser.State, file *parser.File, runExpensive bool) interfaces.PublishDiagnosticsNotification {
	f := file.Snapshot()

	analyses := Analyse(state, file, runExpensive)

	return NewDiagnosticNotification(f.URI, f.Version, DiagnosticsFromAnalyses(analyses))
}

func NewDiagnostic(node *sitter.Node, severity int, source string, message string) interfaces.Diagnostic {
	r := utils.Range{Start: utils.LspPositionFromTsPosition(node.StartPosition()), End: utils.LspPositionFromTsPosition(node.EndPosition())}

	return interfaces.Diagnostic{
		Range:    r,
		Severity: &severity,
		Source:   &source,
		Message:  message,
	}
}

func DiagnosticFromAnalysis(analysis interfaces.Analysis) interfaces.Diagnostic {
	code := any(analysis.Code)

	diagnostic := interfaces.Diagnostic{
		Code:     &code,
		Range:    analysis.Range,
		Severity: &analysis.Severity,
		Source:   &analysis.Source,
		Message:  analysis.Message,
	}

	relatedInformation := []interfaces.DiagnosticRelatedInformation{}
	for _, ri := range analysis.RelatedInformation {
		dri := interfaces.DiagnosticRelatedInformation{
			Location: interfaces.Location{Uri: ri.Uri, Range: ri.Range},
			Message:  ri.Message,
		}
		relatedInformation = append(relatedInformation, dri)
	}

	diagnostic.RelatedInformation = &relatedInformation

	return diagnostic
}

func DiagnosticsFromAnalyses(analyses []interfaces.Analysis) []interfaces.Diagnostic {
	diagnostics := []interfaces.Diagnostic{}

	for _, analysis := range analyses {
		diagnostics = append(diagnostics, DiagnosticFromAnalysis(analysis))
	}

	return diagnostics
}
