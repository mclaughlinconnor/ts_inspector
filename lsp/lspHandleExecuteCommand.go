package lsp

import (
	"log"
	"ts_inspector/commands"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func lspHandleExecuteCommand(writer *utils.Writer, logger *log.Logger, state *parser.State, request interfaces.ExecuteCommandRequest) {
	commandName := request.Params.Command
	args := request.Params.Arguments

	command, ok := commands.CommandMap[commandName]
	if !ok {
		logger.Printf("Error: could not find command %s", commandName)
		return
	}

	changes, err := command.Perform(writer, state, args)
	if err != nil {
		logger.Printf("Error: %s", err)
		return
	}

	if len(changes) == 0 {
		return
	}

	utils.WriteResponse(
		writer,
		interfaces.ApplyWorkspaceEditRequest{
			RequestMessage: interfaces.RequestMessage{
				RPC:    "2.0",
				ID:     utils.GetNextId(),
				Method: "workspace/applyEdit",
			},
			Params: interfaces.ApplyWorkspaceEditParams{
				Label: command.Title,
				Edit:  interfaces.WorkspaceEdit{Changes: changes},
			},
		},
	)
}
