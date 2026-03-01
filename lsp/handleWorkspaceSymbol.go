package lsp

import (
	"io"
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/search"
	"ts_inspector/utils"
)

func newWorkspaceSymbolResponse(id int, symbols []interfaces.WorkspaceSymbol) interfaces.WorkspaceSymbolResponse {
	return interfaces.WorkspaceSymbolResponse{
		Response: interfaces.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: symbols,
	}
}

func HandleWorkspaceSymbol(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.WorkspaceSymbolRequest) {
	params := request.Params

	interestingPoints, err := search.FindInterestingPoints(params.Query)
	if err != nil {
		logger.Println(err)
	}

	symbols := make([]interfaces.WorkspaceSymbol, 0)
	for _, ip := range interestingPoints {
		symbol := interfaces.WorkspaceSymbol{Name: ip.Text, Location: ip.Location, Kind: ip.Kind}
		symbols = append(symbols, symbol)
	}

	utils.WriteResponse(writer, newWorkspaceSymbolResponse(request.ID, symbols))
}
