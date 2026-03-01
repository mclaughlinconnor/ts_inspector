package lsp

import (
	"io"
	"log"
	traversetypescriptfiles "ts_inspector/ast/indexing"
	"ts_inspector/commands"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/search"
	"ts_inspector/utils"
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
				HoverProvider:           true,
				ReferencesProvider:      true,
				TextDocumentSync:        interfaces.TextDocumentSyncKind.Full,
				WorkspaceSymbolProvider: utils.SemanticSearch,
			},
		},
	}
}

func HandleInitialise(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.InitializeRequest) {
	response := newInitializeResponse(request.ID)
	utils.WriteResponse(writer, response)

	state.SetRootUri(request.Params.RootUri)
	utils.WriteResponse(writer, interfaces.BuildMessageNotification("Starting indexing...", interfaces.MessageType.Info))
	filenames := traversetypescriptfiles.Index(state.GetRootPath())

	var err error
	for _, filename := range filenames {
		err = parser.IndexFileFromIndexer(state, filename)
		if err != nil {
			logger.Fatal(err)
		}
	}

	utils.WriteResponse(writer, interfaces.BuildMessageNotification("Postprocessing...", interfaces.MessageType.Info))
	state.Postprocess()

	if utils.SemanticSearch {
		utils.WriteResponse(writer, interfaces.BuildMessageNotification("Building search indexes...", interfaces.MessageType.Info))
		go (func() {
			search.InitSearch()
			search.IndexState(state)
			utils.WriteResponse(writer, interfaces.BuildMessageNotification("Search index built", interfaces.MessageType.Info))
		})()
	}

	utils.WriteResponse(writer, interfaces.BuildMessageNotification("State ready", interfaces.MessageType.Info))
}
