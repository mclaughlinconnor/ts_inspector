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
		for _, file := range state {
			utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(file))
		}
	}

	return state
}
