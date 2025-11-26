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
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	items := make([]interfaces.CompletionItem, 0)
	if file.Snapshot().Filetype != "pug" {
		send(writer, items, &request.ID)

		return
	}

	for _, c := range file.Snapshot().Classes {
		for _, d := range c.GetAllPublicDefinitions() {
			name := d.Name

			item := interfaces.CompletionItem{}

			switch t := d.Node.Type(); t {
			case "method_definition":
				fallthrough
			case "method_signature":
				fallthrough
			case "abstract_method_signature":
				item.Kind = &interfaces.CompletionItemKind.Method
				item.Label = name + "()"

				insertText := name + "($0)"
				item.InsertText = &insertText
				item.InsertTextFormat = &interfaces.InsertTextFormat.Snippet
			case "property_definition": // is this even a thing?
				fallthrough
			case "public_field_definition":
				item.Kind = &interfaces.CompletionItemKind.Property
				item.Label = name
			default:
				// Nothing
			}

			details := interfaces.CompletionItemLabelDetails{
				Description: d.Class.Snapshot().Name,
			}

			item.LabelDetails = &details

			items = append(items, item)
		}

		if !c.HasComponent() {
			continue
		}

		components := c.Snapshot().Angular.Component.GetAvailableComponents()
		for _, c := range components {
			details := interfaces.CompletionItemLabelDetails{
				Description: c.Snapshot().Name,
			}

			selectors := c.Snapshot().Angular.Component.Selectors

			item := interfaces.CompletionItem{
				LabelDetails:     &details,
				Kind:             &interfaces.CompletionItemKind.Class,
				InsertTextFormat: &interfaces.InsertTextFormat.Snippet,
			}

			documentation := c.GetDocumentation(false)
			item.Documentation = &documentation

			for _, selector := range selectors {
				i := interfaces.CompletionItem(item)

				insertText := selector + "($0)"
				i.Label = selector
				i.InsertText = &insertText

				items = append(items, i)
			}
		}
	}

	send(writer, items, &request.ID)
}
