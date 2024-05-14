package lsp

import (
	"io"
	"log"
	"ts_inspector/parser"
)

type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type DidChangeTextDocumentNotification struct {
	Notification
	Params DidChangeTextDocumentNotificationParams `json:"params"`
}

type DidChangeTextDocumentNotificationParams struct {
	TextDocument   TextDocumentItem                 `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

func HandleDidChange(writer io.Writer, logger *log.Logger, state parser.State, request DidChangeTextDocumentNotification) parser.State {
	state, err := parser.HandleFile(state, request.Params.TextDocument.Uri, request.Params.TextDocument.LanguageId, request.Params.TextDocument.Version, logger)

	if err != nil {
		logger.Println(err)
	} else {
		for _, file := range state {
			WriteResponse(writer, GenerateDiagnosticsForFile(file))
		}
	}

	return state
}
