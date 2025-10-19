package lsp

import (
	"bufio"
	"io"
	"log"
	"os"
	"runtime/debug"

	"ts_inspector/actions"
	"ts_inspector/commands"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/rpc"
	"ts_inspector/utils"
)

var Shutdown = make(chan int, 1)

func Start() {
	logger := utils.GetLogger("ts_inspector")
	logger.Println("Started")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)
	big := 1024 * 1024 // 1 mb
	buf := make([]byte, big)
	scanner.Buffer(buf, big)

	writer := os.Stdout

	utils.InitQueries()
	actions.InitActions()
	commands.InitCommands()
	state := parser.State{Classes: map[string]*parser.Class{}, Files: map[string]*parser.File{}}

	for scanner.Scan() {
		logger.Println("Scanner found the next message")
		msg := scanner.Bytes()
		logger.Println("Received msg", msg)
		method, contents, err := rpc.DecodeMessage(msg)
		logger.Println(method)
		if err != nil {
			logger.Printf("Error: %s", err)
			continue
		}

		ns, ok := handleMessage(logger, writer, state, method, contents, msg)
		if ok {
			state = ns
		}
	}

	logger.Println("LSP event loop finished")

	if err := scanner.Err(); err != nil {
		logger.Fatal(err)
	}
}

func handleMessage(logger *log.Logger, writer io.Writer, state parser.State, method string, contents []byte, msg []byte) (parser.State, bool) {
	defer func() (parser.State, bool) {
		if r := recover(); r != nil {
			logger.Println("Panicked with: ", r, "responding with empty response")
			logger.Println("Stack: ", string(debug.Stack()))
			utils.WriteResponse(writer, Response{RPC: "2.0", ID: &utils.MostRecentId})

		}

		return state, false
	}()

	r := utils.TryParseRequest[interfaces.Request](logger, contents)
	utils.MostRecentId = r.ID

	logger.Printf("Received msg with method: %s", method)

	switch method {
	case "initialize":
		request := utils.TryParseRequest[interfaces.InitializeRequest](logger, contents)
		state.RootURI = request.Params.RootUri
		response := HandleInitialise(writer, logger, request)
		utils.WriteResponse(writer, response)
	case "shutdown":
		Shutdown <- 1
	case "textDocument/didOpen":
		request := utils.TryParseRequest[interfaces.DidOpenTextDocumentNotification](logger, contents)
		state = HandleDidOpen(writer, logger, state, request)
	case "textDocument/didChange":
		request := utils.TryParseRequest[interfaces.DidChangeTextDocumentNotification](logger, contents)
		state = HandleDidChange(writer, logger, state, request)
	case "textDocument/codeAction":
		request := utils.TryParseRequest[interfaces.CodeActionRequest](logger, contents)
		HandleCodeAction(writer, logger, state, request)
	case "workspace/executeCommand":
		request := utils.TryParseRequest[interfaces.ExecuteCommandRequest](logger, contents)
		HandleExecuteCommand(writer, logger, state, request)
	case "initialized":
		{
		}
	default:
		if utils.MostRecentId == 0 || method == "" {
			break
		}

		log.Println("Not handling request for:", method)

		utils.WriteResponse(writer, Response{RPC: "2.0", ID: &utils.MostRecentId})
	}

	return state, true
}
