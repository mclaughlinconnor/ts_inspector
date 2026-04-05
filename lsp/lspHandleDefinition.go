package lsp

import (
	"io"
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func lspHandleDefinition(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.DefinitionRequest) {
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	locations := make([]interfaces.Location, 0)
	if file == nil || file.Snapshot().Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

		return
	}

	offset := file.GetOffsetForPosition(request.Params.Position)
	locations = append(locations, parser.FindDefinition(state, file, offset)...)

	utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
}
