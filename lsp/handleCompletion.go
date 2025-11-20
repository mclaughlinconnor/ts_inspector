package lsp

import (
	"io"
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func send(writer io.Writer, items []interfaces.CompletionItem, id *int) {
	utils.WriteResponse(writer, interfaces.CompletionResponse{Result: items, Response: interfaces.Response{ID: id, RPC: "2.0"}})
}

func HandleCompletion(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.CompletionRequest) {
	file := state.Files[parser.FilenameFromUri(request.Params.TextDocument.Uri)]

	items := make([]interfaces.CompletionItem, 0)
	if file.Filetype != "pug" {
		send(writer, items, &request.ID)

		return
	}

	for _, c := range file.Classes {
		for _, d := range c.GetAllPublicDefinitions() {
			items = append(items, interfaces.CompletionItem{Label: d.Name})
		}

		if !c.HasComponent() {
			continue
		}

		tags := c.Angular.Component.GetAvailableTags()
		for _, t := range tags {
			items = append(items, interfaces.CompletionItem{Label: t})
		}
	}

	send(writer, items, &request.ID)
}
