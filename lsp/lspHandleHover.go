package lsp

import (
	"io"
	"log"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

func lspHandleHover(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.HoverRequest) {
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	if file == nil || file.Snapshot().Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

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
					matches, parsed := tagUnderCursor.MatchesSelector(selector)
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

	if utils.TsGo {
		sb = handleTsGoHover(sb, state, file, int(cursorOffset))
	}

	if len(sb) == 0 {
		utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
		return
	}

	hover := interfaces.Hover{Contents: interfaces.MarkupContent{Kind: interfaces.MarkupKind.Markdown, Value: strings.Join(sb, "\n---\n")}}
	utils.WriteResponse(writer, interfaces.HoverResponse{Result: hover, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
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

	v := state.GetTsGo().GetSymbolAtPosition(file.GetTcbUri(), uint32(offset))
	w := state.GetTsGo().GetTypeOfSymbol(v.Result.Id)
	x := state.GetTsGo().TypeToString(w.Result.Id)

	return append(sb, x.Result)
}
