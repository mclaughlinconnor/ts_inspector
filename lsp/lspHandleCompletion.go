package lsp

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

type completionContext struct {
	*context
	items   []interfaces.CompletionItem
	request *interfaces.CompletionRequest
}

func lspHandleCompletion(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.CompletionRequest) {
	context, err := buildCompletionContext(writer, logger, state, &request)
	if err != nil {
		logErrorWithResponse(writer, logger, err, request.ID)
		return
	}

	if context == nil || !context.file.IsPug() {
		emptyResponse(writer, request.ID)
		return
	}

	if context.ci.namedNodeUnderCursor == nil {
		emptyResponse(writer, request.ID)
		return
	}

	nodeType := context.ci.namedNodeUnderCursor.Type()
	for _, thing := range context.file.Components() {
		switch nodeType {
		case "interpolation_content":
			fallthrough
		case "source_file":
			fallthrough
		case "children":
			err = buildCompletionTag(context, thing)
		case "content":
			fallthrough
		case "attribute_value":
			fallthrough
		case "quoted_attribute_value":
			fallthrough
		case "javascript":
			buildCompletionProperty(context, thing)
		case "attributes":
			err = buildCompletionAttribute(context, thing)
		default:
			context.logger.Printf("Tried to trigger completions inside of %v node\n", nodeType)
		}

		if err != nil {
			logErrorWithResponse(writer, logger, err, request.ID)
			return
		}
	}

	utils.WriteResponse(writer, interfaces.CompletionResponse{Result: context.items, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})
}

func buildCompletionAttribute(context *completionContext, class *parser.Class) error {
	for _, thing := range class.GetAvailableThings(context.state) {
		if thing.HasComponent() {
			err := buildCompletionComponent(context, thing)
			if err != nil {
				return err
			}
		}

		if thing.HasDirective() {
			err := buildCompletionDirective(context, thing)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func buildCompletionContext(writer *utils.Writer, logger *log.Logger, state *parser.State, request *interfaces.CompletionRequest) (*completionContext, error) {
	context, err := buildContext(writer, logger, state, request.Params.TextDocument, request.Params.Position)
	if context == nil || err != nil {
		return nil, err
	}

	return &completionContext{
		context: context,
		items:   []interfaces.CompletionItem{},
		request: request,
	}, nil
}

func buildCompletionAngularBinding(context *completionContext, definition parser.ClassedDefinition, openChar string, closeChar string, input bool, output bool) interfaces.CompletionItem {
	name := openChar
	if input {
		name += definition.GetInputName()
	}
	if output {
		name += definition.GetOutputName()
	}
	name += closeChar

	insertText := name + "='$0'"

	return interfaces.CompletionItem{
		InsertText:       &insertText,
		InsertTextFormat: &interfaces.InsertTextFormat.Snippet,
		TextEdit:         &interfaces.TextEdit{Range: context.ci.cursorRange, NewText: insertText},
		Kind:             &interfaces.CompletionItemKind.Property,
		Label:            name,
		Documentation:    &interfaces.MarkupContent{Kind: interfaces.MarkupKind.PlainText, Value: definition.Name + ": " + definition.Type},
		LabelDetails: &interfaces.CompletionItemLabelDetails{
			Description: definition.Class.Snapshot().Name,
		},
	}
}

func buildCompletionComponent(context *completionContext, thing *parser.Class) error {
	matches, _, err := context.ci.tagUnderCursor.MatchesSelectorAny(thing.GetSelectors())
	if err != nil {
		return err
	}

	if !matches {
		return nil
	}

	for _, definition := range thing.GetInputs(true) {
		context.items = append(context.items, buildCompletionAngularBinding(context, definition, "[", "]", true, false))
	}

	for _, definition := range thing.GetOutputs() {
		context.items = append(context.items, buildCompletionAngularBinding(context, definition, "(", ")", false, true))
	}

	return nil
}

func buildCompletionDirective(context *completionContext, thing *parser.Class) error {
	tagUnderCursor := context.ci.tagUnderCursor

	for _, selector := range thing.GetSelectors() {
		parsedSelector, err := ast.ParseSelector(selector)
		if err != nil {
			return err
		}

		if len(parsedSelector.Attributes) == 0 {
			continue
		}

		matchesSelector, _ := tagUnderCursor.MatchesParsedSelector(parsedSelector.WithoutAttributes())
		if !matchesSelector {
			continue
		}

		item := interfaces.CompletionItem{
			Documentation:    &interfaces.MarkupContent{Kind: interfaces.MarkupKind.Markdown, Value: thing.GetDocumentation(true)},
			InsertTextFormat: &interfaces.InsertTextFormat.Snippet,
			Kind:             &interfaces.CompletionItemKind.Property,
			Label:            selector,
			LabelDetails:     &interfaces.CompletionItemLabelDetails{Description: thing.Snapshot().Name},
		}

		insertTextLines := []string{}

		tabStopIndex := 1
		for _, selectorAttribute := range parsedSelector.Attributes {
			if slices.ContainsFunc(tagUnderCursor.Attributes, func(tagAttribute string) bool {
				return utils.StripAngularFromAttributeNoType(tagAttribute) == selectorAttribute
			}) {
				continue
			}

			insertTextLines = append(insertTextLines, fmt.Sprintf("[%v]='$%v'", selectorAttribute, tabStopIndex))
			tabStopIndex++
		}

		insertText := strings.Join(insertTextLines, ", ")

		item.TextEdit = &interfaces.TextEdit{Range: context.ci.cursorRange, NewText: insertText}
		item.InsertText = &insertText

		context.items = append(context.items, item)
	}

	return nil
}

func buildCompletionProperty(context *completionContext, class *parser.Class) {
	for _, d := range class.GetAllPublicDefinitions() {
		item := interfaces.CompletionItem{
			LabelDetails: &interfaces.CompletionItemLabelDetails{
				Description: d.Class.Snapshot().Name,
			},
		}

		propertyName := d.Name
		propertyNodeType := d.Node.Type()

		switch propertyNodeType {
		case "public_field_definition":
			item.Kind = &interfaces.CompletionItemKind.Property
			item.Label = propertyName
		case "method_definition":
			fallthrough
		case "method_signature":
			fallthrough
		case "abstract_method_signature":
			item.Kind = &interfaces.CompletionItemKind.Method
			item.Label = propertyName + "()"

			insertText := propertyName + "($0)"
			item.InsertText = &insertText
			item.InsertTextFormat = &interfaces.InsertTextFormat.Snippet
		}

		item.Documentation = &interfaces.MarkupContent{Kind: interfaces.MarkupKind.PlainText, Value: item.Label + ": " + d.Type}

		context.items = append(context.items, item)
	}
}

func buildCompletionTag(context *completionContext, class *parser.Class) error {
	for _, thing := range class.GetAvailableThings(context.state) {
		if !thing.HasComponent() {
			continue
		}

		rawSelectors := thing.GetSelectors()
		selectors, err := ast.ParseSelectorsArray(rawSelectors)
		if err != nil {
			return err
		}

		baseItem := buildBaseCompletionItem(thing)

		for i, selector := range selectors {
			if selector.Tag == "" {
				continue
			}

			item := interfaces.CompletionItem(baseItem)

			insertText := buildInsertTextFromSelector(selector)
			item.InsertText = &insertText

			item.Label = rawSelectors[i]

			context.items = append(context.items, item)
		}
	}

	return nil
}

func buildInsertTextFromSelector(selector *ast.Selector) string {
	insertText := strings.Builder{}
	insertText.WriteString(selector.Tag)

	insertText.WriteRune('(')
	for i, attribute := range selector.Attributes {
		insertText.WriteRune('[')
		insertText.WriteString(attribute)
		insertText.WriteString("]='$")
		insertText.WriteString(strconv.Itoa(i))
		insertText.WriteRune('\'')
	}
	insertText.WriteString("$0)")

	return insertText.String()
}

func buildBaseCompletionItem(thing *parser.Class) interfaces.CompletionItem {
	return interfaces.CompletionItem{
		Documentation: &interfaces.MarkupContent{
			Kind:  interfaces.MarkupKind.Markdown,
			Value: thing.GetDocumentation(false),
		},
		LabelDetails: &interfaces.CompletionItemLabelDetails{
			Description: thing.Snapshot().Name,
		},
		Kind:             &interfaces.CompletionItemKind.Class,
		InsertTextFormat: &interfaces.InsertTextFormat.Snippet,
	}
}
