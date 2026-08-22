package lsp

import (
	"log"
	"ts_inspector/actions"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func newCodeActionResponse(id int, codeActions []interfaces.CodeAction) interfaces.CodeActionRepsonse {
	return interfaces.CodeActionRepsonse{
		ResponseMessage: interfaces.ResponseMessage{
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

func lspHandleCodeAction(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.CodeActionRequest) {
	file, found := state.GetFile(parser.FilenameFromUri(request.Params.TextDocument.Uri))
	if !found {
		utils.WriteResponse(writer, newCodeActionResponse(request.ID, []interfaces.CodeAction{}))
		return
	}

	codeActions := GenerateActions(writer, logger, state, file, request.Params.Range)

	utils.WriteResponse(writer, newCodeActionResponse(request.ID, codeActions))
}

func GenerateActions(writer *utils.Writer, logger *log.Logger, state *parser.State, file *parser.File, editRange utils.Range) []interfaces.CodeAction {
	codeActions := []interfaces.CodeAction{}

	for _, action := range actions.Actions {
		edits, command, allowed, err := action.Perform(writer, state, file, editRange)

		if err != nil {
			logger.Printf("Error: %s", err)
		}

		if allowed && err == nil && ((edits != nil && len(*edits) > 0) || command != nil) {
			ca := interfaces.CodeAction{
				Title: action.Title,
				Kind:  interfaces.CodeActionKind.Source,
			}

			if edits != nil && len(*edits) > 0 {
				we := WorkspaceEditFromEdits(file, *edits)
				ca.Edit = &we
			}

			if command != nil {
				ca.Command = command
			}

			codeActions = append(codeActions, ca)
		}
	}

	return codeActions
}
