package lsp

import (
	"io"
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func HandleDidChange(writer io.Writer, logger *log.Logger, state parser.State, request interfaces.DidChangeTextDocumentNotification) parser.State {
	state, err := parser.HandleFile(
		state,
		request.Params.TextDocument.Uri,
		request.Params.TextDocument.LanguageId,
		request.Params.TextDocument.Version,
		request.Params.ContentChanges[0].Text,
		logger,
	)

	if err != nil {
		logger.Println(err)
	} else {
		file := state[parser.FilenameFromUri(request.Params.TextDocument.Uri)]

		// My diagnostics only work on files with a controller or template
		if file.Controller == "" && file.Template == "" {
			return state
		}

		utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(file))

		if file.Controller != "" {
			utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(state[file.Controller]))
		}
		if file.Template != "" {
			utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(state[file.Template]))
		}
	}

	return state
}
