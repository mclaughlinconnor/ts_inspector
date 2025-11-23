package lsp

import (
	"io"
	"log"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func HandleReferences(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.ReferenceRequest) {
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	locations := make([]interfaces.Location, 0)
	if file.Snapshot().Filetype != "pug" {
		utils.WriteResponse(writer, interfaces.ReferenceResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})

		return
	}

	offset := file.GetOffsetForPosition(request.Params.Position)

	tagName, found := ast.GetTagNameAtOffset(file.Snapshot().Content, offset)
	if !found {
		utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
	}

	for _, c := range *state.GetClasses() {
		if !c.HasComponent() {
			continue
		}

		template := c.Angular.Component.Template
		if template == nil {
			continue
		}

		usages, found := template.TagUsages[tagName]
		if !found || len(usages.Usages) == 0 {
			continue
		}

		templateFile := c.Angular.Component.TemplateUrlFile

		for _, usage := range usages.Usages {
			tf := templateFile.Snapshot()

			start := parser.GetPositionForOffset(tf.Content, usage.Node.StartByte())
			end := parser.GetPositionForOffset(tf.Content, usage.Node.EndByte())

			locations = append(locations, interfaces.Location{Uri: tf.URI, Range: utils.Range{Start: start, End: end}})
		}
	}

	if request.Params.Context.IncludeDeclaration {
		locations = append(locations, parser.FindDefinition(file, offset)...)
	}

	utils.WriteResponse(writer, interfaces.DefinitionResponse{Result: locations, Response: interfaces.Response{ID: &request.ID, RPC: "2.0"}})
}
