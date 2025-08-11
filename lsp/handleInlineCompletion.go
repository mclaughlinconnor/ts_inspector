package lsp

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
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

	pugContext := ""
	if model == "mellum" {
		pugContext = filename_token + templateFile.Filename() + "\n" +
			fim_suffix_token + utils.Substring(templateFile.Content, int(templateOffset), len(templateFile.Content)) +
			fim_prefix_token + utils.Substring(templateFile.Content, 0, int(templateOffset)) +
			fim_middle_token
	} else if model == "qwen" {
		pugContext = filename_token + templateFile.Filename() + "\n" +
			fim_prefix_token + utils.Substring(templateFile.Content, 0, int(templateOffset)) +
			fim_suffix_token + utils.Substring(templateFile.Content, int(templateOffset), len(templateFile.Content)) +
			fim_middle_token
	}

	return tsContext + pugContext
}

func escape(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(text, "\\", "\\\\"), "\"", "\\\""), "\n", "\\n")
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

	jsonPrompt := "{\"prompt\": \"" + escape(contextfulString) + "\"}"

	resp, err := http.Post("http://localhost:8080/v1/completions", "application/json", bytes.NewBufferString(jsonPrompt))
	if err != nil {
		logger.Print(err)
		return
	}

	defer resp.Body.Close()
	responseString, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Print(err)
		return
	}

	var responseJson interfaces.MLXServerCompletionResult
	err = json.Unmarshal(responseString, &responseJson)
	if err != nil {
		logger.Print(err)
		return
	}

	replaceRange := utils.Range{Start: position, End: position}

	items := make([]interfaces.InlineCompletionItem, 0)
	for _, choice := range responseJson.Choices {
		item := interfaces.InlineCompletionItem{InsertText: choice.Text, Range: &replaceRange}
		items = append(items, item)
	}

	utils.WriteResponse(writer, newInlineCompletionResponse(request.ID, items))
}
