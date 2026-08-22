package lsp

import (
	"log"
	"ts_inspector/analysis"
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func lspHandleDidChange(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.DidChangeTextDocumentNotification) {
	err := parser.IndexFileFromLsp(
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
		file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))
		if file == nil {
			return
		}

		if config.GetConfig().TsGo.Enable {
			state.GetTsGo().UpdateSnapshot("", []parser.DocumentIdentifier{{URI: file.Snapshot().URI}})
		}

		dependencies := file.GetDependencies(state)
		for _, dependency := range dependencies {
			file, found := state.GetFile(dependency)
			if !found {
				continue
			}

			file.Postprocess(state)
		}
		file.Postprocess(state)

		utils.WriteResponse(writer, analysis.GenerateDiagnosticsForFile(state, file, true))

		for _, depFile := range file.GetDependencies(state) {
			if file.Filename() == depFile {
				continue
			}

			file, found := state.GetFile(depFile)
			if !found || !file.Snapshot().IsOpen {
				continue
			}

			utils.WriteResponse(writer, analysis.GenerateDiagnosticsForFile(state, file, false))
		}
	}
}
