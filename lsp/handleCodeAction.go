package lsp

import (
	"io"
	"log"
	"ts_inspector/actions"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func newCodeActionResponse(id int, codeActions []interfaces.CodeAction) interfaces.CodeActionRepsonse {
	return interfaces.CodeActionRepsonse{
		Response: interfaces.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: codeActions,
	}
}

func HandleCodeAction(writer io.Writer, logger *log.Logger, state parser.State, request interfaces.CodeActionRequest) {
	file := state[parser.FilenameFromUri(request.Params.TextDocument.Uri)]

	codeActions := GenerateActions(logger, state, file, request.Params.Range)

	utils.WriteResponse(writer, newCodeActionResponse(request.ID, codeActions))
}

func GenerateActions(logger *log.Logger, state parser.State, file parser.File, editRange utils.Range) []interfaces.CodeAction {
	codeActions := []interfaces.CodeAction{}

	for _, action := range actions.Actions {
		edits, allowed, err := action.Perform(state, file, editRange)

		if err != nil {
			logger.Printf("Error: %s", err)
		}

		if allowed && err == nil && len(edits) > 0 {
			codeActions = append(codeActions, interfaces.CodeAction{
				Title: action.Title,
				Edit:  interfaces.WorkspaceEditFromEdits(file, edits),
			})
		}
	}

	return codeActions
}
