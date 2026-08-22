package lsp

import (
	"log"
	"slices"
	"strconv"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func send(writer *utils.Writer, items []interfaces.CompletionItem, id *int) {
	utils.WriteResponse(writer, interfaces.CompletionResponse{Result: items, ResponseMessage: interfaces.ResponseMessage{ID: id, RPC: "2.0"}})
}

func lspHandleCompletion(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.CompletionRequest) {
	items := make([]interfaces.CompletionItem, 0)

	file, found := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))
	if !found {
		send(writer, items, &request.ID)
		return
	}

	if file.Snapshot().Filetype != "pug" {
		send(writer, items, &request.ID)

		return
	}

	offset := file.GetOffsetForPosition(request.Params.Position)

	root, err := utils.ParseText([]byte(file.Snapshot().Content), utils.Pug)
	node := ast.GetNamedNodeAtPosition(root, offset)

	if err != nil || node == nil {
		send(writer, items, &request.ID)
		notification := interfaces.BuildMessageNotification(err.Error(), interfaces.MessageType.Error)
		utils.WriteResponse(writer, notification)

		logger.Println(err)
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
				is, err := getAttrCompletions(state, file, c, offset)
				if err != nil {
					send(writer, items, &request.ID)
					notification := interfaces.BuildMessageNotification(err.Error(), interfaces.MessageType.Error)
					utils.WriteResponse(writer, notification)

					logger.Println(err)
					return
				}

				items = append(items, is...)
			}
		}
	}

	send(writer, items, &request.ID)
}

func getAttrCompletions(state *parser.State, file *parser.File, class *parser.Class, cursorOffset uint32) ([]interfaces.CompletionItem, error) {
	items := make([]interfaces.CompletionItem, 0)

	if !class.HasComponent() {
		return items, nil
	}

	tag, found := ast.GetTagAtOffset(file.Snapshot().Content, cursorOffset)
	if !found {
		return items, nil
	}

	var err error

	things := class.Snapshot().Angular.Component.GetAvailableThings(state)
	for _, thing := range things {
		if thing.HasComponent() {
			items, err = forComponentThing(thing, file, cursorOffset, &tag, items)
			if err != nil {
				return items, err
			}
		}

		if thing.HasDirective() {
			items = forDirectiveThing(thing, file, cursorOffset, &tag, items)
		}
	}

	return items, nil
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

		if !thing.HasComponent() {
			continue
		}

		item := interfaces.CompletionItem{
			LabelDetails:     &details,
			Kind:             &interfaces.CompletionItemKind.Class,
			InsertTextFormat: &interfaces.InsertTextFormat.Snippet,
		}

		documentation := thing.GetDocumentation(false)
		item.Documentation = &interfaces.MarkupContent{Kind: interfaces.MarkupKind.Markdown, Value: documentation}

		for _, selector := range thing.Snapshot().Angular.Component.Selectors {
			ps, err := ast.ParseSelector(selector)
			if err != nil || len(ps.Tag) == 0 {
				continue
			}

			i := interfaces.CompletionItem(item)

			insertText := strings.Builder{}
			insertText.WriteString(ps.Tag)

			insertText.WriteRune('(')
			for i, psa := range ps.Attributes {
				insertText.WriteRune('[')
				insertText.WriteString(psa)
				insertText.WriteString("]='$")
				insertText.WriteString(strconv.Itoa(i))
				insertText.WriteRune('\'')
			}
			insertText.WriteString("$0)")

			it := insertText.String()
			i.InsertText = &it

			i.Label = selector

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

func forComponentThing(thing *parser.Class, file *parser.File, cursorOffset uint32, tagName *ast.Tag, items []interfaces.CompletionItem) ([]interfaces.CompletionItem, error) {
	matches := false
	for _, s := range thing.Snapshot().Angular.Component.Selectors {
		m, _, err := tagName.MatchesSelector(s)
		if err != nil {
			return items, err
		}
		if m {
			matches = m
			break
		}
	}

	if !matches {
		return items, nil
	}

	cursorPosition := utils.GetPositionForOffset(file.Snapshot().Content, cursorOffset)
	cursorRange := utils.Range{Start: cursorPosition, End: cursorPosition}

	for _, i := range thing.GetInputs(true) {
		items = append(items, build(i, cursorRange, "[", "]", true, false))
	}

	for _, i := range thing.GetOutputs() {
		items = append(items, build(i, cursorRange, "(", ")", false, true))
	}

	return items, nil
}

func forDirectiveThing(thing *parser.Class, file *parser.File, cursorOffset uint32, tag *ast.Tag, items []interfaces.CompletionItem) []interfaces.CompletionItem {
	cursorPosition := utils.GetPositionForOffset(file.Snapshot().Content, cursorOffset)
	cursorRange := utils.Range{Start: cursorPosition, End: cursorPosition}

	for _, selector := range thing.Snapshot().Angular.Directive.Selectors {
		parsedSelector, err := ast.ParseSelector(selector)
		if err != nil {
			continue
		}

		matchesSelector, _ := tag.MatchesParsedSelector(parsedSelector.WithoutAttributes())
		if !matchesSelector {
			continue
		}

		if len(parsedSelector.Attributes) == 0 {
			continue
		}

		item := interfaces.CompletionItem{}

		itBuilder := strings.Builder{}

		pos := 1
		for _, attr := range parsedSelector.Attributes {
			matches := func(attribute string) bool {
				a, _ := utils.StripAngularFromAttribute(attribute)
				return a == attr
			}

			if slices.ContainsFunc(tag.Attributes, matches) {
				continue
			}

			if pos > 1 {
				itBuilder.WriteString(", ")
			}

			itBuilder.WriteString("[")
			itBuilder.WriteString(attr)
			itBuilder.WriteString("]='$" + strconv.Itoa(pos) + "'")

			pos++
		}

		insertText := itBuilder.String()

		item.InsertText = &insertText
		item.InsertTextFormat = &interfaces.InsertTextFormat.Snippet

		// Can't use insertText because clients can do post-processing on the text, which can lead to tag((output)) losing some brackets
		textEdit := interfaces.TextEdit{}
		textEdit.Range = cursorRange
		textEdit.NewText = insertText
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
