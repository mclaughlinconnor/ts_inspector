package analysis

import (
	"ts_inspector/utils"
)

type Analysis struct {
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
