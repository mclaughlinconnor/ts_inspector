package interfaces

import (
	"ts_inspector/utils"
)

type CodeDescription struct {
	Href string `json:"href"`
}

type Diagnostic struct {
	Range utils.Range `json:"range"`

	Severity *int `json:"severity"`

	Code *any `json:"code"`

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
