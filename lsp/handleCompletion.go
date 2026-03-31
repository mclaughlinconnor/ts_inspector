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
				items = append(items, getTagCompletions(state, c)...)
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
				items = append(items, getAttrCompletions(state, file, c, offset)...)
			}
		}
	}

	send(writer, items, &request.ID)
}

func getAttrCompletions(state *parser.State, file *parser.File, class *parser.Class, cursorOffset uint32) []interfaces.CompletionItem {
	items := make([]interfaces.CompletionItem, 0)

	if !class.HasComponent() {
		return items
	}

	tagName, found := ast.GetTagAtOffset(file.Snapshot().Content, cursorOffset)
	if !found {
		return items
	}

	things := class.Snapshot().Angular.Component.GetAvailableThings(state)
	for _, thing := range things {
		if thing.HasComponent() {
			items = forComponentThing(thing, file, cursorOffset, &tagName, items)
		}

		if thing.HasDirective() {
			items = forDirectiveThing(thing, file, cursorOffset, &tagName, items)
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

func getTagCompletions(state *parser.State, class *parser.Class) []interfaces.CompletionItem {
	items := make([]interfaces.CompletionItem, 0)

	if !class.HasComponent() {
		return items
	}

	things := class.Snapshot().Angular.Component.GetAvailableThings(state)
	for _, thing := range things {
		details := interfaces.CompletionItemLabelDetails{
			Description: thing.Snapshot().Name,
		}

		selectors := []string{}
		if thing.HasComponent() {
			selectors = append(selectors, thing.Snapshot().Angular.Component.Selectors...)
		}

		if thing.HasDirective() {
			selectors = append(selectors, thing.Snapshot().Angular.Directive.Selectors...)
		}

		item := interfaces.CompletionItem{
			LabelDetails:     &details,
			Kind:             &interfaces.CompletionItemKind.Class,
			InsertTextFormat: &interfaces.InsertTextFormat.Snippet,
		}

		documentation := thing.GetDocumentation(false)
		item.Documentation = &interfaces.MarkupContent{Kind: interfaces.MarkupKind.Markdown, Value: documentation}

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

func build(definition parser.ClassedDefinition, cursorRange utils.Range, openChar string, closeChar string, input bool, output bool) interfaces.CompletionItem {
	item := interfaces.CompletionItem{}

	name := openChar
	if input {
		name += definition.GetInputName()
	}
	if output {
		name += definition.GetOutputName()
	}
	name += closeChar

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

func forComponentThing(thing *parser.Class, file *parser.File, cursorOffset uint32, tagName *ast.Tag, items []interfaces.CompletionItem) []interfaces.CompletionItem {
	matches := false
	for _, s := range thing.Snapshot().Angular.Component.Selectors {
		m, _ := tagName.MatchesSelector(s)
		if m {
			matches = m
			break
		}
	}

	if !matches {
		return items
	}

	cursorPosition := utils.GetPositionForOffset(file.Snapshot().Content, cursorOffset)
	cursorRange := utils.Range{Start: cursorPosition, End: cursorPosition}

	for _, i := range thing.GetInputs() {
		items = append(items, build(i, cursorRange, "[", "]", true, false))
	}

	for _, i := range thing.GetOutputs() {
		items = append(items, build(i, cursorRange, "(", ")", false, true))
	}

	return items
}

func forDirectiveThing(thing *parser.Class, file *parser.File, cursorOffset uint32, tagName *ast.Tag, items []interfaces.CompletionItem) []interfaces.CompletionItem {
	cursorPosition := utils.GetPositionForOffset(file.Snapshot().Content, cursorOffset)
	cursorRange := utils.Range{Start: cursorPosition, End: cursorPosition}

	for _, selector := range thing.Snapshot().Angular.Directive.Selectors {
		valid, _, attr := ast.ExtractTagNameAndAttrFromSelector(selector)
		if !valid || attr == "" {
			continue
		}

		item := interfaces.CompletionItem{}

		insertText := selector + "='$0'"
		item.InsertText = &insertText
		item.InsertTextFormat = &interfaces.InsertTextFormat.Snippet

		// Can't use insertText because clients can do post-processing on the text, which can lead to tag((output)) losing some brackets
		textEdit := interfaces.TextEdit{}
		textEdit.Range = cursorRange
		textEdit.NewText = selector + "='$0'"
		item.TextEdit = &textEdit

		item.Kind = &interfaces.CompletionItemKind.Property
		item.Label = selector

		documentation := interfaces.MarkupContent{Kind: interfaces.MarkupKind.Markdown, Value: thing.GetDocumentation(true)}
		item.Documentation = &documentation

		details := interfaces.CompletionItemLabelDetails{
			Description: thing.Snapshot().Name,
		}

		item.LabelDetails = &details

		items = append(items, item)
	}

	return items
}
