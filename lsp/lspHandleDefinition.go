package lsp

import (
	"io"
	"log"
	"strings"
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

func lspHandleDefinition(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.DefinitionRequest) {
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	locations := make([]interfaces.Location, 0)
	if file == nil || file.Snapshot().Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

		return
	}

	offset := file.GetOffsetForPosition(request.Params.Position)
	locations = append(locations, parser.FindDefinition(state, file, offset)...)

	if config.TsGo {
		part := tcb.PugToTsLocation(state, file, int(offset), int(offset))

		if part != nil {
			v := state.GetTsGo().GetSymbolAtPosition(file.GetTcbUri(), uint32(*part.TsStartOffset))
		DECLARATION:
			for _, declaration := range v.Result.Declarations {
				node, err := declaration.ExtractNode()
				if err != nil {
					continue
				}

				if strings.HasPrefix(node.Path, "bundled") {
					continue
				}

				// Variables can be declared in template files
				declarationFile, found := state.GetFile(node.Path)
				if !found {
					continue
				}

				for _, class := range declarationFile.Snapshot().Classes {
					definitions := class.FilterOwnDefinitions(nodeFilter(node.Pos, node.End))
					for _, definition := range definitions {
						locations = append(locations, definition.GetLocation())
					}

					if len(definitions) > 0 {
						continue DECLARATION
					}
				}

				locations = append(locations, declarationFile.GetLocationForOffset(uint32(node.Pos+1), uint32(node.End+1)))
			}
		}
	}

	utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
}

func nodeFilter(offsetStart int, offsetEnd int) func(parser.ClassedDefinition) bool {
	return func(d parser.ClassedDefinition) bool {
		nodeStart := d.Node.StartByte() + d.Class.Snapshot().Node.StartByte()
		nodeEnd := d.Node.StartByte() + d.Class.Snapshot().Node.StartByte()

		return offsetStart <= int(nodeStart) && offsetEnd >= int(nodeEnd)
	}
}
