package lsp

import (
	"io"
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func HandleDefinition(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.DefinitionRequest) {
	file := state.Files[parser.FilenameFromUri(request.Params.TextDocument.Uri)]

	locations := make([]interfaces.Location, 0)
	if file.Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

		return
	}

	offset := file.GetOffsetForPosition(request.Params.Position)
	locations = append(locations, parser.FindDefinition(file, offset)...)

	utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
}
