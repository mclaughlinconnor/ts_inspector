package lsp

import (
	"io"
	"log"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
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

	offset := file.GetOffsetForPosition(request.Params.Position)

	node, err := utils.ParseFile(false, file.Snapshot().Content, utils.Pug, nil, func(root *sitter.Node, content []byte, v *sitter.Node) (*sitter.Node, error) {
		return ast.GetNamedNodeAtPosition(root, offset), nil
	})

	if err != nil || node == nil {
		return
	}

	for _, c := range file.Snapshot().Classes {
		switch nodeType := node.Type(); nodeType {
		case "interpolation_content":
			fallthrough
		case "source_file":
			fallthrough
		case "children":
			{
				items = append(items, getTagCompletions(c)...)
			}
		case "content":
			fallthrough
		case "attribute_value":
			fallthrough
		case "quoted_attribute_value":
			fallthrough
		case "javascript":
			{
				items = append(items, getPropertyCompletions(c)...)
			}
		case "attributes":
			{
				items = append(items, getAttrCompletions(file, c, offset)...)
			}
		}
	}

	send(writer, items, &request.ID)
}

func getAttrCompletions(file *parser.File, class *parser.Class, cursorOffset uint32) []interfaces.CompletionItem {
	items := make([]interfaces.CompletionItem, 0)

	if !class.HasComponent() {
		return items
	}

	tagName, found := ast.GetNameOfTagAtOffset(file.Snapshot().Content, cursorOffset)
	if !found {
		return items
	}

	components := class.Snapshot().Angular.Component.GetAvailableComponents()
	for _, c := range components {
		if !c.HasComponent() {
			continue
		}

		matches := false
		for _, s := range c.Snapshot().Angular.Component.Selectors {
			matches = matches || s == tagName
		}

		if !matches {
			continue
		}

		cursorPosition := parser.GetPositionForOffset(file.Snapshot().Content, cursorOffset)
		cursorRange := utils.Range{Start: cursorPosition, End: cursorPosition}

		build := func(definition parser.ClassedDefinition, openChar string, closeChar string) interfaces.CompletionItem {
			item := interfaces.CompletionItem{}

			name := openChar + definition.Name + closeChar
			insertText := name + "='$0'"
			item.InsertText = &insertText
			item.InsertTextFormat = &interfaces.InsertTextFormat.Snippet

			// Can't use insertText because clients can do post-processing on the text, which can lead to tag((output)) losing some brackets
			textEdit := interfaces.TextEdit{}
			textEdit.Range = cursorRange
			textEdit.NewText = name + "='$0'"
			item.TextEdit = &textEdit

			item.Kind = &interfaces.CompletionItemKind.Property
			item.Label = name

			documentation := interfaces.MarkupContent{Kind: interfaces.MarkupKind.PlainText, Value: definition.Name + ": " + definition.Type}
			item.Documentation = &documentation

			details := interfaces.CompletionItemLabelDetails{
				Description: definition.Class.Snapshot().Name,
			}

			item.LabelDetails = &details

			return item
		}

		for _, i := range c.GetInputs() {
			items = append(items, build(i, "[", "]"))
		}

		for _, i := range c.GetOutputs() {
			items = append(items, build(i, "(", ")"))
		}
	}

	return items
}

func getPropertyCompletions(class *parser.Class) []interfaces.CompletionItem {
	items := make([]interfaces.CompletionItem, 0)
	for _, d := range class.GetAllPublicDefinitions() {
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

		documentation := interfaces.MarkupContent{Kind: interfaces.MarkupKind.PlainText, Value: item.Label + ": " + d.Type}
		item.Documentation = &documentation

		item.LabelDetails = &details

		items = append(items, item)
	}

	return items
}

func getTagCompletions(class *parser.Class) []interfaces.CompletionItem {
	items := make([]interfaces.CompletionItem, 0)

	if !class.HasComponent() {
		return items
	}

	components := class.Snapshot().Angular.Component.GetAvailableComponents()
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

	return items
}
