package interfaces

import (
	"ts_inspector/parser"
	"ts_inspector/utils"
)

type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	Range utils.Range `json:"range"`

	Context CodeActionContext `json:"context"`
}

type CodeActionRequest struct {
	Request

	Params CodeActionParams `json:"params"`
}

type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type codeActionKind struct {
	Empty                 string
	QuickFix              string
	Refactor              string
	RefactorExtract       string
	RefactorInline        string
	RefactorRewrite       string
	Source                string
	SourceOrganizeImports string
	SourceFixAll          string
}

var CodeActionKind = codeActionKind{"", "quickfix", "refactor", "refactor.extract", "refactor.inline", "refactor.rewrite", "source", "source.organizeImports", "source.fixAll"}

type CodeAction struct {
	Title string `json:"title"`

	Edit WorkspaceEdit `json:"edit"`

  Kind string `json:"kind"`
}

type CodeActionRepsonse struct {
	Response

	Result []CodeAction `json:"result"`
}

type WorkspaceEdit struct {
	Changes map[string]utils.TextEdits `json:"changes"`
}

func WorkspaceEditFromEdits(file parser.File, edits utils.TextEdits) WorkspaceEdit {
	filename := parser.UriFromFilename(file.Filename())
	return WorkspaceEdit{
		Changes: map[string]utils.TextEdits{
			filename: edits,
		},
	}
}
