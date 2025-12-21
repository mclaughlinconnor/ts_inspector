package interfaces

import (
	"ts_inspector/utils"
)

type CodeDescription struct {
	Href string `json:"href"`
}

type Diagnostic struct {
	Range utils.Range `json:"range"`

	Severity *int `json:"severity,omitempty"`

	Code *any `json:"code,omitempty"`

	CodeDescription *CodeDescription `json:"codeDescription"`

	Source *string `json:"source"`

	Message string `json:"message"`

	Tags *[]int `json:"tags,omitempty"`

	RelatedInformation *[]DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`

	Data *any `json:"data,omitempty"`
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
