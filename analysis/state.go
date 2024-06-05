package analysis

import sitter "github.com/smacker/go-tree-sitter"

// uri: analysis
var CurrentAnalysis = map[string][]Analysis{}

type Analysis struct {
	Node *sitter.Node

	Severity int

	Source string

	Message string
}

type severity struct {
	Error int

	Warning int

	Information int

	Hint int
}

var AnalysisSeverity = severity{1, 2, 3, 4}
