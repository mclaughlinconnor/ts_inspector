package lsp

import (
	"io"
	"log"
	"ts_inspector/commands"
	"ts_inspector/interfaces"
)

func newInitializeResponse(id int) interfaces.InitializeResponse {
	triggerChars := []string{".", "\"", "'", "`", "/", "@", "<", "#", " ", "*"}

	for i := 'A'; i <= 'z'; i++ {
		triggerChars = append(triggerChars, string(i))
	}

	commitChars := []string{".", ",", ";"}

	labelDetailsSupport := true

	return interfaces.InitializeResponse{
		Response: interfaces.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: interfaces.InitializeResult{
			Capabilities: interfaces.ServerCapabilities{
				CodeActionProvider: true,
				CompletionProvider: interfaces.CompletionOptions{
					AllCommitCharacters: &commitChars,
					CompletionItem: &struct {
						LabelDetailsSupport *bool `json:"labelDetailsSupport,omitempty"`
					}{LabelDetailsSupport: &labelDetailsSupport},
					TriggerCharacters: &triggerChars,
				},
				DefinitionProvider: true,
				ExecuteCommandProvider: interfaces.ExecuteCommandOptions{
					Commands: commands.CommandNames,
				},
				HoverProvider:      true,
				ReferencesProvider: true,
				TextDocumentSync:   interfaces.TextDocumentSyncKind.Full,
			},
		},
	}
}

func HandleInitialise(writer io.Writer, logger *log.Logger, request interfaces.InitializeRequest) interfaces.InitializeResponse {
	msg := newInitializeResponse(request.ID)
	return msg
}
