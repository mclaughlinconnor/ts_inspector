package interfaces

import (
	"ts_inspector/analysis"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type CodeDescription struct {
	Href string `json:"href"`
}

type Diagnostic struct {
	Range utils.Range `json:"range"`

	Severity *int `json:"severity"`

	Code *int `json:"code"`

	CodeDescription *CodeDescription `json:"codeDescription"`

	Source *string `json:"source"`

	Message string `json:"message"`

	Tags *[]int `json:"tags"`

	RelatedInformation *[]DiagnosticRelatedInformation `json:"relatedInformation"`

	Data *any `json:"data"`
}

type diagnosticTag struct {
	Unnecessary int

	Deprecated int
}

var DiagnosticTag = diagnosticTag{1, 2}

type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`

	Message string `json:"message"`
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

	Version *int `json:"version,omitempty"`

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
		Params: PublishDiagnosticsParams{uri, &version, diagnostics},
	}
}

func GenerateDiagnosticsForFile(file parser.File) PublishDiagnosticsNotification {
	return NewDiagnosticNotification(file.URI, file.Version, DiagnosticsFromAnalyses(analysis.Analyse(file)))
}

func NewDiagnostic(node *sitter.Node, severity int, source string, message string) Diagnostic {
	r := utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}

	return Diagnostic{
		Range:    r,
		Severity: &severity,
		Source:   &source,
		Message:  message,
	}
}

func DiagnosticsFromAnalysis(analysis analysis.Analysis) Diagnostic {
	r := utils.Range{Start: utils.PositionFromPoint(analysis.Node.StartPoint()), End: utils.PositionFromPoint(analysis.Node.EndPoint())}

	return Diagnostic{
		Range:    r,
		Severity: &analysis.Severity,
		Source:   &analysis.Source,
		Message:  analysis.Message,
	}
}

func DiagnosticsFromAnalyses(analyses []analysis.Analysis) []Diagnostic {
	diagnostics := []Diagnostic{}

	for _, analysis := range analyses {
		r := utils.Range{Start: utils.PositionFromPoint(analysis.Node.StartPoint()), End: utils.PositionFromPoint(analysis.Node.EndPoint())}
		diagnostics = append(diagnostics, Diagnostic{
			Range:    r,
			Severity: &analysis.Severity,
			Source:   &analysis.Source,
			Message:  analysis.Message,
		})
	}

	return diagnostics
}
