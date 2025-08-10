package lsp

import (
	"io"
	"log"
	"net/http"
	"ts_inspector/interfaces"
	"ts_inspector/utils"
)

func newInlineCompletionResponse(id int, completions []interfaces.InlineCompletionItem) interfaces.InlineCompletionResponse {
	return interfaces.InlineCompletionResponse{
		Response: interfaces.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: completions,
	}
}

func HandleInlineCompletion(writer io.Writer, logger *log.Logger, request interfaces.InlineCompletionRequest) {
	resp, err := http.Get("http://localhost:8088/completions.txt")
	if err != nil {
		logger.Print(err)
		return
	}

	defer resp.Body.Close()
	results, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Print(err)
		return
	}

	item := interfaces.InlineCompletionItem{InsertText: string(results)}
	items := make([]interfaces.InlineCompletionItem, 1)
	items = append(items, item)

	utils.WriteResponse(writer, newInlineCompletionResponse(request.ID, items))
}
