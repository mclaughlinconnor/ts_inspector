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

func HandleDefinition(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.DefinitionRequest) {
	file := state.Files[parser.FilenameFromUri(request.Params.TextDocument.Uri)]

	locations := make([]interfaces.Location, 0)
	if file.Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

		return
	}

	offset := file.GetOffsetForPosition(request.Params.Position)

	tagName, _ := utils.ParseFile(false, file.Content, utils.Pug, "", func(root *sitter.Node, content []byte, v string) (string, error) {
		node := ast.HasNodeInHierarchy(root, "tag_name", offset, offset)
		if node == nil {
			return "", nil
		}

		tagName := node.Content([]byte(file.Content))
		return tagName, nil
	})

	if tagName != "" {
		for _, c := range file.Classes {
			if !c.HasComponent() {
				continue
			}

			components := c.Angular.Component.GetAvailableComponents()
			for _, c := range components {
				if c.Angular.Component.Selector == tagName {
					start := parser.GetPositionForOffset(c.File.Content, c.NameNode.StartByte()+c.Node.StartByte())
					end := parser.GetPositionForOffset(c.File.Content, c.NameNode.EndByte()+c.Node.StartByte())

					locations = append(locations, interfaces.Location{Uri: c.File.URI, Range: utils.Range{Start: start, End: end}})
				}
			}
		}
	}

	utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
}
