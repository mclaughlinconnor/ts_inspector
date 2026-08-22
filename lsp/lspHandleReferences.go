package lsp

import (
	"log"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func lspHandleReferences(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.ReferenceRequest) {
	locations := make([]interfaces.Location, 0)

	file, found := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))
	if !found {
		utils.WriteResponse(writer, interfaces.ReferenceResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})

		return
	}

	if file.Snapshot().Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.ReferenceResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})

		return
	}

	offset := file.GetOffsetForPosition(request.Params.Position)

	// TODO: needs to find usages for every selector on a component

	tagName, found := ast.GetTagNameAtOffset(file.Snapshot().Content, offset)
	if found {
		for _, c := range state.GetClasses() {
			if !c.HasComponent() {
				continue
			}

			template := c.Snapshot().Angular.Component.Template
			if template == nil {
				continue
			}

			usages, found := template.TagUsages[tagName]
			if !found || len(usages.Usages) == 0 {
				continue
			}

			templateFile := c.Snapshot().Angular.Component.TemplateUrlFile

			for _, usage := range usages.Usages {
				tf := templateFile.Snapshot()

				start := utils.GetPositionForOffset(tf.Content, usage.Node.StartByte())
				end := utils.GetPositionForOffset(tf.Content, usage.Node.EndByte())

				locations = append(locations, interfaces.Location{Uri: tf.URI, Range: utils.Range{Start: start, End: end}})
			}
		}
	}

	if request.Params.Context.IncludeDeclaration {
		ls, err := parser.FindDefinition(state, file, offset)
		if err != nil {
			utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})
			notification := interfaces.BuildMessageNotification(err.Error(), interfaces.MessageType.Error)
			utils.WriteResponse(writer, notification)

			logger.Println(err)
			return
		}

		locations = append(locations, ls...)
	}

	utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, ResponseMessage: interfaces.ResponseMessage{ID: &request.ID, RPC: "2.0"}})
}
