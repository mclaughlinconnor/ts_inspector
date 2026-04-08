package lsp

import (
	"io"
	"log"
	"net/url"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb_cm"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func lspHandleTcb(writer io.Writer, logger *log.Logger, state *parser.State, request interfaces.TcbRequest) {
	parsedUrl, err := url.Parse(request.Params.Uri)
	if err != nil {
		response := interfaces.TcbRequestResponse{Response: interfaces.Response{RPC: "2.0", ID: &request.ID}, Result: err.Error()}
		utils.WriteResponse(writer, response)
		return
	}

	parsedUrl.Path = strings.TrimSuffix(parsedUrl.Path, interfaces.TCB_FILENAME_SUFFIX) + ".pug"
	fileUrl := parsedUrl.String()

	file, _ := state.GetFile(parser.FilenameFromUri(fileUrl))
	if file == nil || file.Snapshot().Filetype != "pug" {
		response := interfaces.TcbRequestResponse{Response: interfaces.Response{RPC: "2.0", ID: &request.ID}, Result: ""}
		utils.WriteResponse(writer, response)
		return
	}

	content := []byte(file.Snapshot().Content)
	tcbBlock, err := utils.ParseText(content, utils.Pug, "", func(root *sitter.Node, _ []byte, _ string) (string, error) {
		tcb := tcb_cm.GenerateTcb(state, file.Snapshot().Classes[0], root, content)

		return tcb.ToString(), nil
	})

	if err != nil {
		tcbBlock = err.Error()
	}

	response := interfaces.TcbRequestResponse{
		Response: interfaces.Response{RPC: "2.0", ID: &request.ID},
		Result:   tcbBlock,
	}

	utils.WriteResponse(writer, response)
}
