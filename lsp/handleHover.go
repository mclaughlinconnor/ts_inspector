package lsp

import (
	// "fmt"
	"io"
	"log"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	// aast "ts_inspector/parser/ast"
	"ts_inspector/utils"
	// sitter "github.com/smacker/go-tree-sitter"
)

func HandleHover(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.HoverRequest) {
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	if file == nil || file.Snapshot().Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

		return
	}

	// utils.ParseFile(false, file.Snapshot().Content, utils.Pug, nil, func(root *sitter.Node, content []byte, edits any) (any, error) {
	// 	current := &aast.Node{Kind: aast.KindRoot, Children: []*aast.Node{}, Attributes: []*aast.Node{}, Name: "", Value: ""}
	// 	state := aast.Ast{Children: []*aast.Node{current}, Content: content, Current: current}
	//
	// 	aast.InitAstParser()
	// 	aast.Parse(&state, root)
	//
	// 	fmt.Printf("%+v\n", state)
	//
	// 	return nil, nil
	// })

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
					if tagUnderCursor.MatchesSelector(selector) {
						if thing.HasComponent() {
							sb = handleAttributeHover(sb, thing, attributeName, selector)
						}

						if thing.HasDirective() {
							sb = handleTagHover(sb, thing)
						}

					} else if selector == attributeName { // component with `selector: '[formControl]`
						sb = handleTagHover(sb, thing)
					}
				}
			}
		}
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

	// TODO: get directives here

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
