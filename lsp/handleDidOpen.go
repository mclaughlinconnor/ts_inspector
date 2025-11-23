package lsp

import (
	"io"
	"log"
	"ts_inspector/analysis"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func HandleDidOpen(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.DidOpenTextDocumentNotification) {
	err := parser.IndexFileFromLsp(
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
		file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))
		if file == nil {
			return
		}

		dependencies := file.GetDependencies(state)
		for _, dependency := range dependencies {
			file, _ := state.GetFile(dependency)
			file.Postprocess(state)
		}
		file.Postprocess(state)

		utils.WriteResponse(writer, analysis.GenerateDiagnosticsForFile(file))

		for _, depFile := range file.GetDependencies(state) {
			file, _ := state.GetFile(depFile)
			utils.WriteResponse(writer, analysis.GenerateDiagnosticsForFile(file))
		}
	}
}
