package ngserver

import (
	"os"
	"ts_inspector/analysis"
	"ts_inspector/commands"
	"ts_inspector/interfaces"
	"ts_inspector/rpc"
	"ts_inspector/utils"
)

type RequestData struct {
	Method   string
	Position *uint32
}

var Requests = map[int]RequestData{}

func HandleResponse(method string, contents []byte, msg []byte) {
	var writer = os.Stdout

	if method == "" {
		response := utils.TryParseRequest[interfaces.Request](logger, contents)
		method = Requests[response.ID].Method
	}

	switch method {
	case "textDocument/completion":
		response := utils.TryParseRequest[interfaces.CompletionResponse](logger, contents)
		data, found := Requests[*response.ID]
		if found {
			for i := range len(response.Result) {
				response.Result[i].Data["CM_Position"] = data.Position
			}
		}
		utils.WriteResponse(writer, response)
	case "textDocument/publishDiagnostics":
		// TOOD: cache diagnostics so I can resend them as part of ts_inspector's analysis
		response := utils.TryParseRequest[interfaces.PublishDiagnosticsNotification](logger, contents)
		tsInspectorDiagnostics := interfaces.DiagnosticsFromAnalyses(analysis.CurrentAnalysis[response.Params.Uri])
		response.Params.Diagnostics = append(response.Params.Diagnostics, tsInspectorDiagnostics...)
		utils.WriteResponse(writer, response)
	case "initialize":
		response := utils.TryParseRequest[interfaces.InitializeResponse](logger, contents)
		response.Result.Capabilities.TextDocumentSync = interfaces.TextDocumentSyncKind.Full
		response.Result.Capabilities.CodeActionProvider = true
		response.Result.Capabilities.ExecuteCommandProvider = interfaces.ExecuteCommandOptions{Commands: commands.GetLspCommands()}
		nmsg := rpc.EncodeMessage(response)
		writer.Write([]byte(nmsg))
	default:
		writer.Write(msg)
	}
}
