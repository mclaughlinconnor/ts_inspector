package lsp

import (
	"io"
	"log"
	"strings"
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

		components := c.Angular.Component.GetAvailableComponents()
		for _, c := range components {
			details := interfaces.CompletionItemLabelDetails{
				Description: c.Name,
			}

			selector := c.Angular.Component.Selector
			insertText := selector + "($0)"

			item := interfaces.CompletionItem{
				Label:            selector,
				LabelDetails:     &details,
				Kind:             &interfaces.CompletionItemKind.Class,
				InsertTextFormat: &interfaces.InsertTextFormat.Snippet,
				InsertText:       &insertText,
			}

			if len(c.Angular.Component.DeclaredIn) > 0 {
				modules := make([]string, 0)

				for _, d := range c.Angular.Component.DeclaredIn {
					modules = append(modules, d.Name)
				}

				str := "Declared in: " + strings.Join(modules, ", ")

				item.Documentation = &str
			}

			items = append(items, item)
		}
	}

	send(writer, items, &request.ID)
}
