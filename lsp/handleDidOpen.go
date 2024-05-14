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
	Params DidOpenTextDocumentNotificationParams `json:"params"`
}

type DidOpenTextDocumentNotificationParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

func HandleDidOpen(writer io.Writer, logger *log.Logger, state parser.State, request DidOpenTextDocumentNotification) parser.State {
	state, err := parser.HandleFile(
		state,
		request.Params.TextDocument.Uri,
		request.Params.TextDocument.LanguageId,
		request.Params.TextDocument.Version,
		"", // no ContentChanges
		logger,
	)

	if err != nil {
		logger.Println(err)
	} else {
		for _, file := range state {
			WriteResponse(writer, GenerateDiagnosticsForFile(file))
		}
	}

	return state
}
