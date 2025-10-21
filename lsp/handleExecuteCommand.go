package lsp

import (
	"io"
	"log"
	"math/rand"
	"ts_inspector/commands"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func response(writer io.Writer, id int) {
	utils.WriteResponse(writer, Response{RPC: "2.0", ID: &id})
}

func HandleExecuteCommand(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.ExecuteCommandRequest) {
	commandName := request.Params.Command
	args := request.Params.Arguments

	command, ok := commands.CommandMap[commandName]
	if !ok {
		logger.Printf("Error: could not find command %s", commandName)
		response(writer, request.ID)
		return
	}

	changes, err := command.Perform(state, args)
	if err != nil {
		logger.Printf("Error: %s", err)
		response(writer, request.ID)
		return
	}

	response(writer, request.ID)

	utils.WriteResponse(
		writer,
		interfaces.ApplyWorkspaceEditRequest{
			Request: interfaces.Request{
				RPC:    "2.0",
				ID:     rand.Intn(10_000),
				Method: "workspace/applyEdit",
			},
			Params: interfaces.ApplyWorkspaceEditParams{
				Label: command.Title,
				Edit:  interfaces.WorkspaceEdit{Changes: changes},
			},
		},
	)
}
