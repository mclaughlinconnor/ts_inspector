package lsp

import (
	"io"
	"log"
	"ts_inspector/parser"
)

type TextDocumentContentChangeEvent struct {
  Text string `json:"text"`;
};

type DidChangeTextDocumentNotification struct {
	Notification
	TextDocument TextDocumentItem `json:"textDocument"`
  ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

func HandleDidChange(writer io.Writer, logger *log.Logger, state parser.State, request DidChangeTextDocumentNotification) parser.State {
	if request.TextDocument.LanguageId == "typescript" {
		state, _ = parser.HandleTypeScriptFile(request.TextDocument.Uri)
	} else if request.TextDocument.LanguageId == "pug" {
		state, _ = parser.HandlePugFile(request.TextDocument.Uri, state)
	}

	return state
}
