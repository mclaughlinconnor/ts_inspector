package lsp

import (
	"log"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

type hoverContext struct {
	*context
	request *interfaces.HoverRequest
	sb      []string
}

func lspHandleHover(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.HoverRequest) {
	context := buildHoverContext(writer, logger, state, &request)
	if context == nil || !context.file.IsPug() {
		emptyResponse(writer, request.ID)
		return
	}

THING:
	for _, thing := range context.file.GetAvailableThings(context.state) {
		matchingSelectors, err := thing.SelectorsMatchingTag(context.ci.tagUnderCursor)
		if err != nil {
			logErrorWithResponse(writer, logger, err, request.ID)
			return
		}

		for _, selector := range matchingSelectors {
			if handleSelector(context, thing, selector) {
				continue THING
			}
		}
	}

	if config.GetConfig().TsGo.Enable {
		err := buildTsGoHoverDocumentation(context)
		if err != nil {
			logErrorWithResponse(writer, logger, err, request.ID)
		}
	}

	if len(context.sb) == 0 {
		utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})
		return
	}

	hover := interfaces.Hover{Contents: interfaces.MarkupContent{Kind: interfaces.MarkupKind.Markdown, Value: strings.Join(context.sb, "\n---\n")}}
	utils.WriteResponse(writer, interfaces.HoverResponse{Result: hover, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})
}

func buildAttrHoverDocumentation(context *hoverContext, thing *parser.Class, selector *ast.Selector) bool {
	attributeName := context.ci.attributeUnderCursor
	handled := false

	for _, definition := range thing.FilterAllDefinitions(func(def parser.ClassedDefinition) bool { return def.NameMatchesString(attributeName) }) {
		context.sb = append(context.sb, definition.GetDocumentation(true))
		handled = true
	}

	if selector.MatchesAttribute(context.ci.attributeUnderCursor) { // component with `selector: '[formControl]`
		handled = handled || buildTagHoverDocumentation(context, thing)
	}

	if !strings.HasPrefix(attributeName, "[") && !strings.HasSuffix(attributeName, "]") { // attribute with *ngIf
		handled = handled || buildTagHoverDocumentation(context, thing)
	}

	return handled
}

func buildHoverContext(writer *utils.Writer, logger *log.Logger, state *parser.State, request *interfaces.HoverRequest) *hoverContext {
	context := buildContext(writer, logger, state, request.Params.TextDocument, request.Params.Position)
	if context == nil {
		return nil
	}

	return &hoverContext{context: context, request: request, sb: []string{}}
}

func buildTagHoverDocumentation(context *hoverContext, thing *parser.Class) bool {
	context.sb = append(context.sb, thing.GetDocumentation(true))

	return true
}

func buildTsGoHoverDocumentation(context *hoverContext) error {
	tcb, err := tcb.BuildTcbBlock(context.state, context.file)
	if err != nil {
		return err
	}

	cursorOffset := context.ci.cursorOffset

	part := tcb.PugToTsLocation(cursorOffset, cursorOffset)
	if part == nil {
		return nil
	}

	cursorOffsetFromStartOfPart := cursorOffset - *part.PugStartOffset
	offset := *part.TsStartOffset + cursorOffsetFromStartOfPart

	tcbUri, err := context.file.GetTcbUri()
	if err != nil {
		return err
	}

	ttype := context.state.GetTsGo().GetTypeAtPosition(tcbUri, offset)
	if ttype == nil {
		return nil
	}

	text := context.state.GetTsGo().TypeToString(ttype.Result.Id)
	if text == nil {
		return nil
	}

	context.sb = append(context.sb, text.Result)

	return nil
}

func handleSelector(context *hoverContext, thing *parser.Class, selector *ast.Selector) bool {
	handled := false

	if context.ci.isOnTagName {
		handled = handled || buildTagHoverDocumentation(context, thing)
	}

	if context.ci.isOnAttrName {
		handled = handled || buildAttrHoverDocumentation(context, thing, selector)
	}

	return handled
}
