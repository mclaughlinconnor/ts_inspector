package lsp

import (
	"io"
	"log"
	"net/http"
	"path"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
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

func buildLLMContext(templateFile parser.File, controllerFile parser.File, position utils.Position) string {
	model := "qwen"

	var filename_token = ""
	var fim_suffix_token = ""
	var fim_prefix_token = ""
	var fim_middle_token = ""

	if model == "mellum" {
		filename_token = "<filename>"
		fim_suffix_token = "<fim_suffix>"
		fim_prefix_token = "<fim_prefix>"
		fim_middle_token = "<fim_middle>"
	} else if model == "qwen" {
		filename_token = "<|file_sep|>"
		fim_suffix_token = "<|fim_suffix|>"
		fim_prefix_token = "<|fim_prefix|>"
		fim_middle_token = "<|fim_middle|>"
	}

	templateOffset := templateFile.GetOffsetForPosition(position)

	tsContext := ""
	if len(controllerFile.Content) != 0 {
		tsContext = filename_token + controllerFile.Filename() + "\n" + controllerFile.Content + "\n"
	}

	pugContext := filename_token + templateFile.Filename() + "\n" +
		fim_prefix_token + utils.Substring(templateFile.Content, 0, int(templateOffset)) +
		fim_suffix_token + utils.Substring(templateFile.Content, int(templateOffset), len(templateFile.Content)) +
		fim_middle_token

	return tsContext + pugContext
}

func HandleInlineCompletion(writer io.Writer, logger *log.Logger, state parser.State, request interfaces.InlineCompletionRequest) {
	filename := parser.FilenameFromUri(request.Params.TextDocument.Uri)
	if path.Ext(filename) != ".pug" {
		utils.WriteResponse(writer, newInlineCompletionResponse(request.ID, make([]interfaces.InlineCompletionItem, 0)))
		return
	}

	templateFile := state.Files[filename]
	controllerFile := state.Files[templateFile.Controller]
	position := request.Params.Position

	contextfulString := buildLLMContext(templateFile, controllerFile, position)
	logger.Println(contextfulString)

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
	items := make([]interfaces.InlineCompletionItem, 0)
	items = append(items, item)

	utils.WriteResponse(writer, newInlineCompletionResponse(request.ID, items))
}
