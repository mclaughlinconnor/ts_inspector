package lsp

import (
	"io"
	"log"
)

type InitializeRequest struct {
	Request
}

type InitializeResponse struct {
	Response
	Result InitializeResult `json:"result"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	CodeActionProvider bool `json:"codeActionProvider"`
	TextDocumentSync   int  `json:"textDocumentSync"`
}

func newInitializeResponse(id int) InitializeResponse {
	return InitializeResponse{
		Response: Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: InitializeResult{
			Capabilities: ServerCapabilities{
				CodeActionProvider: true,
				TextDocumentSync:   TextDocumentSyncKind.Full,
			},
		},
	}
}

func HandleInitialise(writer io.Writer, logger *log.Logger, request InitializeRequest) {
	msg := newInitializeResponse(request.ID)
	WriteResponse(writer, msg)
}
