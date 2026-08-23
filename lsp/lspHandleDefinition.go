package lsp

import (
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

type definitionContext struct {
	*context
	locations []interfaces.Location
	request   *interfaces.DefinitionRequest
}

func lspHandleDefinition(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.DefinitionRequest) {
	baseContext, context, err := buildDefinitionContext(writer, logger, state, &request)
	if err != nil {
		logErrorWithArrayResponse(writer, logger, err, request.ID)
		return
	}

	if baseContext == nil || context == nil || !context.file.IsPug() {
		emptyArrayResponse(writer, request.ID)
		return
	}

	locations, err := findDefinition(baseContext)
	if err != nil {
		logErrorDefinitionsResponse(context, err)
		return
	}

	context.locations = locations

	definitionsReponse(context)
}

func buildDefinitionContext(writer *utils.Writer, logger *log.Logger, state *parser.State, request *interfaces.DefinitionRequest) (*context, *definitionContext, error) {
	context, err := buildContext(writer, logger, state, request.Params.TextDocument, request.Params.Position)
	if err != nil {
		return nil, nil, err
	}

	if context == nil {
		return nil, nil, nil
	}

	return context, &definitionContext{context: context, locations: []interfaces.Location{}, request: request}, nil
}

func definitionsReponse(context *definitionContext) {
	utils.WriteResponse(context.writer, interfaces.DefinitionResponse{Result: context.locations, ResponseMessage: interfaces.ResponseMessage{ID: &context.request.ID, RPC: "2.0"}})
}

func logErrorDefinitionsResponse(context *definitionContext, err error) {
	logError(context.writer, context.logger, err)
	definitionsReponse(context)
}
