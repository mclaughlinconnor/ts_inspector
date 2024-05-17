package lsp

import (
	"io"
	"log"
	"ts_inspector/actions"
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

type CodeAction struct {
	Title string `json:"title"`

	Edit WorkspaceEdit `json:"edit"`
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

func newCodeActionResponse(id int, codeActions []CodeAction) CodeActionRepsonse {
	return CodeActionRepsonse{
		Response: Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: codeActions,
	}
}

func HandleCodeAction(writer io.Writer, logger *log.Logger, state parser.State, request CodeActionRequest) {
	file := state[parser.FilenameFromUri(request.Params.TextDocument.Uri)]

	codeActions := []CodeAction{}

	onInit, allowed, err := actions.ImplementAngularOnInit(state, file)

	if err != nil {
		logger.Printf("Error: %s", err)
	}

	if allowed && err == nil {
		codeActions = append(codeActions, CodeAction{
			Title: "Add OnInit",
			Edit:  WorkspaceEditFromEdits(file, onInit),
		})
	}

	WriteResponse(writer, newCodeActionResponse(request.ID, codeActions))
}
