package ngserver

import (
	"os"
	"ts_inspector/analysis"
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
		utils.WriteResponse(writer, response)
	case "textDocument/publishDiagnostics":
		tsInspectorDiagnostics := interfaces.DiagnosticsFromAnalyses(analysis.CurrentAnalysis)
		response := utils.TryParseRequest[interfaces.PublishDiagnosticsParams](logger, contents)
		response.Diagnostics = append(response.Diagnostics, tsInspectorDiagnostics...)
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
