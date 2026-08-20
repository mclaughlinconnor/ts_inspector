package lsp

import (
	"io"
	"log"
	"net/url"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

func lspHandleTcb(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.TcbRequest) {
	throwErr := func(err error) {
		response := interfaces.TcbRequestResponse{ResponseMessage: interfaces.ResponseMessage{RPC: "2.0", ID: &request.ID}, Result: err.Error()}
		utils.WriteResponse(writer, response)
	}

	parsedUrl, err := url.Parse(request.Params.Uri)
	if err != nil {
		throwErr(err)
		return
	}

	parsedUrl.Path = strings.TrimSuffix(parsedUrl.Path, interfaces.TCB_FILENAME_SUFFIX) + ".pug"
	fileUrl := parsedUrl.String()

	file, _ := state.GetFile(parser.FilenameFromUri(fileUrl))
	if file == nil || file.Snapshot().Filetype != "pug" || len(file.Snapshot().Classes) < 1 {
		response := interfaces.TcbRequestResponse{ResponseMessage: interfaces.ResponseMessage{RPC: "2.0", ID: &request.ID}, Result: ""}
		utils.WriteResponse(writer, response)
		return
	}

	content := []byte(file.Snapshot().Content)
	root, err := utils.ParseText(content, utils.Pug)
	if err != nil {
		throwErr(err)
		return
	}

	tcb := tcb.GenerateTcb(state, file.Snapshot().Classes[0], root, content)
	tcbBlock := tcb.ToString()

	response := interfaces.TcbRequestResponse{
		ResponseMessage: interfaces.ResponseMessage{RPC: "2.0", ID: &request.ID},
		Result:          tcbBlock,
	}

	utils.WriteResponse(writer, response)
}
