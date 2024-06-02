package ngserver

import (
	"os"
	"ts_inspector/interfaces"
	"ts_inspector/rpc"
	"ts_inspector/utils"
)

var Requests = map[int]string{}

func HandleResponse(method string, contents []byte, msg []byte) {
	var writer = os.Stdout

	response := utils.TryParseRequest[interfaces.Request](logger, contents)
	m := Requests[response.ID]

	switch m {
	case "textDocument/completion":
		response := utils.TryParseRequest[interfaces.CompletionResponse](logger, contents)
		response.Result = []interfaces.CompletionItem{{
			Label: "ts_inspector",
		}}
		utils.WriteResponse(writer, response)
	case "initialize":
		response := utils.TryParseRequest[interfaces.InitializeResponse](logger, contents)
		response.Result.Capabilities.TextDocumentSync = interfaces.TextDocumentSyncKind.Full
		nmsg := rpc.EncodeMessage(response)
		writer.Write([]byte(nmsg))
	default:
		writer.Write(msg)
	}
}
