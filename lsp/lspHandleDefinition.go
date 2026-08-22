package lsp

import (
	"log"
	"strings"
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

func lspHandleDefinition(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.DefinitionRequest) {
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	locations := make([]interfaces.Location, 0)
	if file == nil || file.Snapshot().Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})

		return
	}

	offset := file.GetOffsetForPosition(request.Params.Position)

	ls, err := parser.FindDefinition(state, file, offset)
	if err != nil {
		utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})

		notification := interfaces.BuildMessageNotification(err.Error(), interfaces.MessageType.Error)
		utils.WriteResponse(writer, notification)

		logger.Println(err)

		return
	}

	locations = append(locations, ls...)

	if config.GetConfig().TsGo.Enable {
		part := tcb.PugToTsLocation(state, file, int(offset), int(offset))

		if part != nil {
			tcbUri, err := file.GetTcbUri()
			if err != nil {
				utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})

				notification := interfaces.BuildMessageNotification(err.Error(), interfaces.MessageType.Error)
				utils.WriteResponse(writer, notification)

				logger.Println(err)

				return
			}

			v := state.GetTsGo().GetSymbolAtPosition(tcbUri, uint32(*part.TsStartOffset))
			if v == nil {
				utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})

				return
			}

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

	utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})
}

func nodeFilter(offsetStart int, offsetEnd int) func(parser.ClassedDefinition) bool {
	return func(d parser.ClassedDefinition) bool {
		nodeStart := d.Node.StartByte() + d.Class.Snapshot().Node.StartByte()
		nodeEnd := d.Node.StartByte() + d.Class.Snapshot().Node.StartByte()

		return offsetStart <= int(nodeStart) && offsetEnd >= int(nodeEnd)
	}
}
