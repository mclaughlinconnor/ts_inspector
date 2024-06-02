package interfaces

import (
	"ts_inspector/analysis"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Diagnostic struct {
	Range utils.Range `json:"range"`

	Severity int `json:"severity"`

	Source string `json:"source"`

	Message string `json:"message"`

	// Interesting:
	// Tags []DiagnosticTag
	// RelatedInformation []DiagnosticRelatedInformation
}

type diagnosticSeverity struct {
	Error int

	Warning int

	Information int

	Hint int
}

var DiagnosticSeverity = diagnosticSeverity{1, 2, 3, 4}

type PublishDiagnosticsParams struct {
	Uri string `json:"uri"`

	Version int `json:"version"`

	Diagnostics []Diagnostic `json:"diagnostics"`
}

type PublishDiagnosticsNotification struct {
	Notification
	Params PublishDiagnosticsParams `json:"params"`
}

func NewDiagnosticNotification(uri string, version int, diagnostics []Diagnostic) PublishDiagnosticsNotification {
	return PublishDiagnosticsNotification{
		Notification: Notification{
			RPC:    "2.0",
			Method: "textDocument/publishDiagnostics",
		},
		Params: PublishDiagnosticsParams{uri, version, diagnostics},
	}
}

func GenerateDiagnosticsForFile(file parser.File) PublishDiagnosticsNotification {
	return NewDiagnosticNotification(file.URI, file.Version, DiagnosticsFromAnalyses(analysis.Analyse(file)))
}

func NewDiagnostic(node *sitter.Node, severity int, source string, message string) Diagnostic {
	r := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}

	return Diagnostic{r, severity, source, message}
}

func DiagnosticsFromAnalysis(analysis analysis.Analysis) Diagnostic {
	r := utils.Range{Start: utils.PositionFromPoint(analysis.Node.StartPoint()), End: utils.PositionFromPoint(analysis.Node.EndPoint())}

	return Diagnostic{r, analysis.Severity, analysis.Source, analysis.Message}
}

func DiagnosticsFromAnalyses(analyses []analysis.Analysis) []Diagnostic {
	diagnostics := []Diagnostic{}

	for _, analysis := range analyses {
		r := utils.Range{Start: utils.PositionFromPoint(analysis.Node.StartPoint()), End: utils.PositionFromPoint(analysis.Node.EndPoint())}
		diagnostics = append(diagnostics, Diagnostic{r, analysis.Severity, analysis.Source, analysis.Message})
	}

	return diagnostics
}
