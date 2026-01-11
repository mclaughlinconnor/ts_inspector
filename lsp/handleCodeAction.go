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

func WorkspaceEditFromEdits(file *parser.File, edits utils.TextEdits) interfaces.WorkspaceEdit {
	filename := parser.UriFromFilename(file.Filename())
	return interfaces.WorkspaceEdit{
		Changes: map[string]utils.TextEdits{
			filename: edits,
		},
	}
}

func HandleCodeAction(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.CodeActionRequest) {
	file, _ := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))

	codeActions := GenerateActions(writer, logger, state, file, request.Params.Range)

	utils.WriteResponse(writer, newCodeActionResponse(request.ID, codeActions))
}

func GenerateActions(writer io.Writer, logger *log.Logger, state *parser.State, file *parser.File, editRange utils.Range) []interfaces.CodeAction {
	codeActions := []interfaces.CodeAction{}

	for _, action := range actions.Actions {
		edits, allowed, err := action.Perform(writer, state, file, editRange)

		if err != nil {
			logger.Printf("Error: %s", err)
		}

		if allowed && err == nil && len(edits) > 0 {
			codeActions = append(codeActions, interfaces.CodeAction{
				Edit:  WorkspaceEditFromEdits(file, edits),
				Title: action.Title,
				Kind:  interfaces.CodeActionKind.Source,
			})
		}
	}

	return codeActions
}
