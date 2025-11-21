package lsp

import (
	"io"
	"log"
	"ts_inspector/ast"
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

	tagName, found := ast.GetTagNameAtOffset(file.Content, offset)
	if found {
		for _, c := range file.Classes {
			if !c.HasComponent() {
				continue
			}

			components := c.Angular.Component.GetAvailableComponents()
			for _, c := range components {
				if c.Angular.Component.Selector == tagName {
					start := parser.GetPositionForOffset(c.File.Content, c.NameNode.StartByte()+c.Node.StartByte())
					end := parser.GetPositionForOffset(c.File.Content, c.NameNode.EndByte()+c.Node.StartByte())

					locations = append(locations, interfaces.Location{Uri: c.File.URI, Range: utils.Range{Start: start, End: end}})
				}
			}
		}
	}

	utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
}
