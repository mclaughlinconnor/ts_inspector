package lsp

import (
	"io"
	"log"
	traversetypescriptfiles "ts_inspector/ast/indexing"
	"ts_inspector/commands"
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/search"
	"ts_inspector/utils"

	"golang.org/x/sync/errgroup"
)

func newInitializeResponse(id int) interfaces.InitializeResponse {
	triggerChars := []string{".", "\"", "'", "`", "/", "@", "<", "#", " ", "*"}

	for i := 'A'; i <= 'z'; i++ {
		triggerChars = append(triggerChars, string(i))
	}

	commitChars := []string{".", ",", ";"}

	labelDetailsSupport := true

	return interfaces.InitializeResponse{
		ResponseMessage: interfaces.ResponseMessage{
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
				WorkspaceSymbolProvider: config.SemanticSearch,
			},
		},
	}
}

func lspHandleInitialise(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.InitializeRequest) {
	response := newInitializeResponse(request.ID)
	progressToken, _ := lspCreateProgressToken(writer)
	utils.WriteResponse(writer, response)

	lspBeginProgress(writer, progressToken, "Initialising", "", -1)

	state.SetRootUri(request.Params.RootUri)
	lspReportProgress(writer, progressToken, "Indexing", -1)
	filenames, tsconfigFiles := traversetypescriptfiles.Index(state.GetRootPath())
	state.SetTsConfigFiles(tsconfigFiles)

	if config.IndexingExperiementalParallelInitialIndexing {
		eg := errgroup.Group{}

		for _, filename := range filenames {
			eg.Go(func() error { return parser.IndexFileFromIndexer(state, filename, false) })
		}

		if err := eg.Wait(); err != nil {
			logger.Fatal(err)
		}
	} else {
		var err error
		for _, filename := range filenames {
			err = parser.IndexFileFromIndexer(state, filename, false)
			if err != nil {
				logger.Fatal(err)
			}
		}
	}

	lspReportProgress(writer, progressToken, "Postprocessing", -1)
	state.Postprocess()

	if config.SemanticSearch {
		lspReportProgress(writer, progressToken, "Building search indexes", -1)
		search.InitSearch()
		search.IndexState(state)
	}

	initTsGo(state, writer, progressToken)

	lspEndProgress(writer, progressToken, "Initialised")

	lspReady = true
}

func initTsGo(state *parser.State, writer io.Writer, progressToken *interfaces.ProgressToken) {
	if !config.TsGo {
		return
	}

	lspReportProgress(writer, progressToken, "Starting TsGo", -1)

	t, err := parser.StartTsGo(state)
	if err != nil {
		state.Logger.Print(err)
	}

	state.SetTsGo(t)

	t.Initialize()
	us := t.UpdateSnapshot("tsconfig.json", nil)
	print(us)

	state.SetTsGo(t)
}
