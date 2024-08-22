package lsp

import (
	"io"
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func HandleDidOpen(writer io.Writer, logger *log.Logger, state parser.State, request interfaces.DidOpenTextDocumentNotification) parser.State {
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
		file := state.Files[parser.FilenameFromUri(request.Params.TextDocument.Uri)]

		// My diagnostics only work on files with a controller or template
		if file.Controller == "" && file.Template == "" {
			return state
		}

		utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(file))

		if file.Controller != "" {
			utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(state.Files[file.Controller]))
		}
		if file.Template != "" {
			utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(state.Files[file.Template]))
		}
	}

	return state
}
