package lsp

import (
	"io"
	"log"
	"ts_inspector/parser"
)

type textDocumentSyncKind struct {
	None        int
	Full        int
	Incremental int
}

var TextDocumentSyncKind = textDocumentSyncKind{0, 1, 2}

type DidOpenTextDocumentNotification struct {
	Notification
	TextDocument TextDocumentItem `json:"textDocument"`
}

func HandleDidOpen(writer io.Writer, logger *log.Logger, state parser.State, request DidOpenTextDocumentNotification) parser.State {
	if request.TextDocument.LanguageId == "typescript" {
		state, _ = parser.HandleTypeScriptFile(request.TextDocument.Uri)
	} else if request.TextDocument.LanguageId == "pug" {
		state, _ = parser.HandlePugFile(request.TextDocument.Uri, state)
	}

	return state
}
