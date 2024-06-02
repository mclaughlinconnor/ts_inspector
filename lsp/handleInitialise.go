package lsp

import (
	"io"
	"log"
	"ts_inspector/interfaces"
)

func newInitializeResponse(id int) interfaces.InitializeResponse {
	return interfaces.InitializeResponse{
		Response: interfaces.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: interfaces.InitializeResult{
			Capabilities: interfaces.ServerCapabilities{
				CodeActionProvider: true,
				CompletionProvider: interfaces.CompletionOptions{},
				TextDocumentSync:   interfaces.TextDocumentSyncKind.Full,
			},
		},
	}
}

func HandleInitialise(writer io.Writer, logger *log.Logger, request interfaces.InitializeRequest) interfaces.InitializeResponse {
	msg := newInitializeResponse(request.ID)
	return msg
}
