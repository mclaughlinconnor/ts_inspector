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

	RelatedInformation []RelatedInformation

	Severity int

	Source string
}

type RelatedInformation struct {
	Message string
	Uri     string
	Range   utils.Range
}

type severity struct {
	Error int

	Warning int

	Information int

	Hint int
}

var AnalysisSeverity = severity{1, 2, 3, 4}

func AnalysisSeverityFromTsGoCategory(category *parser.Category) int {
	switch *category {
	case parser.CategoryWarning:
		return AnalysisSeverity.Warning
	case parser.CategoryError:
		return AnalysisSeverity.Error
	case parser.CategorySuggestion:
		return AnalysisSeverity.Hint
	case parser.CategoryMessage:
		return AnalysisSeverity.Information
	default:
		return AnalysisSeverity.Error
	}
}

func NewDiagnosticNotification(uri string, version int, diagnostics []interfaces.Diagnostic) interfaces.PublishDiagnosticsNotification {
	return interfaces.PublishDiagnosticsNotification{
		Notification: interfaces.Notification{
			RPC:    "2.0",
			Method: "textDocument/publishDiagnostics",
		},
		Params: interfaces.PublishDiagnosticsParams{Uri: uri, Version: &version, Diagnostics: diagnostics},
	}
}

func GenerateDiagnosticsForFile(state *parser.State, file *parser.File, runExpensive bool) interfaces.PublishDiagnosticsNotification {
	f := file.Snapshot()

	analyses := Analyse(state, file, runExpensive)

	return NewDiagnosticNotification(f.URI, f.Version, DiagnosticsFromAnalyses(analyses))
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

func DiagnosticsFromAnalyses(analyses []Analysis) []interfaces.Diagnostic {
	diagnostics := []interfaces.Diagnostic{}

	for _, analysis := range analyses {
		diagnostics = append(diagnostics, DiagnosticFromAnalysis(analysis))
	}

	return diagnostics
}
