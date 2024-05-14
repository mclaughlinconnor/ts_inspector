package lsp

import sitter "github.com/smacker/go-tree-sitter"

type Diagnostic struct {
	Range Range `json:"range"`

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

func NewDiagnostic(node *sitter.Node, severity int, source string, message string) Diagnostic {
	r := Range{PositionFromPoint(node.StartPoint()), PositionFromPoint(node.EndPoint())}

	return Diagnostic{r, severity, source, message}
}
