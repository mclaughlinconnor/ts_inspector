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

func lspHandleHover(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.HoverRequest) {
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	if file == nil || file.Snapshot().Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})

		return
	}

	cursorOffset := file.GetOffsetForPosition(request.Params.Position)

	tagName, cursorOnTagName := ast.GetTagNameAtOffset(file.Snapshot().Content, cursorOffset)

	var tagUnderCursor ast.Tag
	attributeName, cursorOnAttributeName := ast.GetAttributeNameAtOffset(file.Snapshot().Content, cursorOffset)
	if cursorOnAttributeName {
		tagUnderCursor, _ = ast.GetTagAtOffset(file.Snapshot().Content, cursorOffset)
	}

	sb := []string{}

LOOP:
	for _, c := range file.Snapshot().Classes {
		if !c.HasComponent() {
			continue
		}

		things := c.Snapshot().Angular.Component.GetAvailableThings(state)
		for _, thing := range things {

			selectors := []string{}
			if thing.HasComponent() {
				selectors = append(selectors, thing.Snapshot().Angular.Component.Selectors...)
			}

			if thing.HasDirective() {
				selectors = append(selectors, thing.Snapshot().Angular.Directive.Selectors...)
			}

			for _, selector := range selectors {
				if cursorOnTagName && selector == tagName {
					sb = handleTagHover(sb, thing)
				}

				if cursorOnAttributeName {
					matches, parsed, err := tagUnderCursor.MatchesSelector(selector)
					if err != nil {
						notification := interfaces.BuildMessageNotification(err.Error(), interfaces.MessageType.Error)
						utils.WriteResponse(writer, notification)

						logger.Println(err)
						break LOOP
					}

					if !matches {
						continue
					}

					sb = handleAttributeHover(sb, thing, attributeName, selector)

					if parsed.MatchesAttribute(attributeName) { // component with `selector: '[formControl]`
						sb = handleTagHover(sb, thing)
					}
				}
			}
		}
	}

	if config.GetConfig().TsGo.Enable {
		sb = handleTsGoHover(sb, state, file, int(cursorOffset))
	}

	if len(sb) == 0 {
		utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})
		return
	}

	hover := interfaces.Hover{Contents: interfaces.MarkupContent{Kind: interfaces.MarkupKind.Markdown, Value: strings.Join(sb, "\n---\n")}}
	utils.WriteResponse(writer, interfaces.HoverResponse{Result: hover, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})
}

func handleAttributeHover(sb []string, class *parser.Class, attributeName string, selector string) []string {
	for _, definition := range class.FilterAllDefinitions(func(def parser.ClassedDefinition) bool { return def.NameMatchesString(attributeName) }) {
		sb = append(sb, definition.GetDocumentation(true))
	}

	if !strings.HasPrefix(attributeName, "[") && !strings.HasSuffix(attributeName, "]") {
		valid, _, attrName := ast.ExtractTagNameAndAttrFromSelector(selector)
		if valid && attrName == attributeName {
			sb = append(sb, class.GetDocumentation(true))
		}
	}

	return sb
}

func handleTagHover(sb []string, class *parser.Class) []string {
	return append(sb, (class.GetDocumentation(true)))
}

func handleTsGoHover(sb []string, state *parser.State, file *parser.File, cursorOffset int) []string {
	tcb, err := tcb.BuildTcbBlock(state, file)
	if tcb == nil || err != nil {
		return sb
	}

	part := tcb.PugToTsLocation(int(cursorOffset), int(cursorOffset))
	if part == nil {
		return sb
	}

	cursorOffsetFromStartOfPart := int(cursorOffset) - *part.PugStartOffset
	offset := *part.TsStartOffset + cursorOffsetFromStartOfPart

	ttype := state.GetTsGo().GetTypeAtPosition(file.GetTcbUri(), uint32(offset))
	if ttype == nil {
		return sb
	}

	text := state.GetTsGo().TypeToString(ttype.Result.Id)
	if text == nil {
		return sb
	}

	return append(sb, text.Result)
}
