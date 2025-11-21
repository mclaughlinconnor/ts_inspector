package analysis

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Analysis struct {
	Code string

	Message string

	Range utils.Range

	Severity int

	Source string
}

type severity struct {
	Error int

	Warning int

	Information int

	Hint int
}

var AnalysisSeverity = severity{1, 2, 3, 4}

func NewDiagnosticNotification(uri string, version int, diagnostics []interfaces.Diagnostic) interfaces.PublishDiagnosticsNotification {
	return interfaces.PublishDiagnosticsNotification{
		Notification: interfaces.Notification{
			RPC:    "2.0",
			Method: "textDocument/publishDiagnostics",
		},
		Params: interfaces.PublishDiagnosticsParams{Uri: uri, Version: &version, Diagnostics: diagnostics},
	}
}

func GenerateDiagnosticsForFile(file parser.File) interfaces.PublishDiagnosticsNotification {
	return NewDiagnosticNotification(file.URI, file.Version, DiagnosticsFromAnalyses(Analyse(file)))
}

func NewDiagnostic(node *sitter.Node, severity int, source string, message string) interfaces.Diagnostic {
	r := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}

	return interfaces.Diagnostic{
		Range:    r,
		Severity: &severity,
		Source:   &source,
		Message:  message,
	}
}

func DiagnosticFromAnalysis(analysis Analysis) interfaces.Diagnostic {
	code := any(analysis.Code)

	return interfaces.Diagnostic{
		Code:     &code,
		Range:    analysis.Range,
		Severity: &analysis.Severity,
		Source:   &analysis.Source,
		Message:  analysis.Message,
	}
}

func DiagnosticsFromAnalyses(analyses []Analysis) []interfaces.Diagnostic {
	diagnostics := []interfaces.Diagnostic{}

	for _, analysis := range analyses {
		diagnostics = append(diagnostics, DiagnosticFromAnalysis(analysis))
	}

	return diagnostics
}
