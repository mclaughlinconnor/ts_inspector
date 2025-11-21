package lsp

import (
	"io"
	"log"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func HandleHover(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.HoverRequest) {
	file := state.Files[parser.FilenameFromUri(request.Params.TextDocument.Uri)]

	if file.Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

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
				if c.Angular.Component.Selector != tagName {
					continue
				}

				markup := c.GetDocumentation(true)
				hover := interfaces.Hover{Contents: markup}

				utils.WriteResponse(writer, interfaces.HoverResponse{Result: hover, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

				return
			}
		}
	}

	utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
}
