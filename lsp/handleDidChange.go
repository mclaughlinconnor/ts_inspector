package lsp

import (
	"io"
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func HandleDidChange(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.DidChangeTextDocumentNotification) {
	err := parser.HandleFile(
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
		file := state.Files[parser.FilenameFromUri(request.Params.TextDocument.Uri)]
		file.PostprocessClasses(state)

		utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(*file))

		for _, depFile := range file.GetDependencies() {
			utils.WriteResponse(writer, interfaces.GenerateDiagnosticsForFile(*state.Files[depFile]))
		}
	}
}
