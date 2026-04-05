package lsp

import (
	"bufio"
	"io"
	"log"
	"os"
	"runtime/debug"

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

	state := parser.CreateState()

	for scanner.Scan() {
		logger.Println("Scanner found the next message")
		msg := scanner.Bytes()
		logger.Println("Received msg", string(msg))
		method, contents, err := rpc.DecodeMessage(msg)
		logger.Println(method)
		if err != nil {
			logger.Printf("Error: %s", err)
			continue
		}

		handleMessage(logger, writer, &state, method, contents)
	}

	logger.Println("LSP event loop finished")

	if err := scanner.Err(); err != nil {
		logger.Fatal(err)
	}
}

func handleMessage(logger *log.Logger, writer io.Writer, state *parser.State, method string, contents []byte) {
	defer func() {
		if r := recover(); r != nil {
			logger.Println("Panicked with: ", r, "responding with empty response")
			logger.Println("Stack: ", string(debug.Stack()))
		}
	}()

	r := utils.TryParseRequest[interfaces.Request](logger, contents)
	utils.MostRecentId = r.ID

	logger.Printf("Received msg with method: %s", method)

	switch method {
	case "initialize":
		request := utils.TryParseRequest[interfaces.InitializeRequest](logger, contents)
		HandleInitialise(writer, logger, state, request)
	case "shutdown":
		Shutdown <- 1
	case "textDocument/didOpen":
		request := utils.TryParseRequest[interfaces.DidOpenTextDocumentNotification](logger, contents)
		HandleDidOpen(writer, logger, state, request)
	case "textDocument/didChange":
		request := utils.TryParseRequest[interfaces.DidChangeTextDocumentNotification](logger, contents)
		HandleDidChange(writer, logger, state, request)
	case "textDocument/codeAction":
		request := utils.TryParseRequest[interfaces.CodeActionRequest](logger, contents)
		HandleCodeAction(writer, logger, state, request)
	case "textDocument/completion":
		request := utils.TryParseRequest[interfaces.CompletionRequest](logger, contents)
		HandleCompletion(writer, logger, state, request)
	case "textDocument/definition":
		request := utils.TryParseRequest[interfaces.DefinitionRequest](logger, contents)
		HandleDefinition(writer, logger, state, request)
	case "textDocument/hover":
		request := utils.TryParseRequest[interfaces.HoverRequest](logger, contents)
		HandleHover(writer, logger, state, request)
	case "textDocument/references":
		request := utils.TryParseRequest[interfaces.ReferenceRequest](logger, contents)
		HandleReferences(writer, logger, state, request)
	case "workspace/executeCommand":
		request := utils.TryParseRequest[interfaces.ExecuteCommandRequest](logger, contents)
		HandleExecuteCommand(writer, logger, state, request)
	case "workspace/symbol":
		request := utils.TryParseRequest[interfaces.WorkspaceSymbolRequest](logger, contents)
		HandleWorkspaceSymbol(writer, logger, state, request)
	case "ts_inspector/getTcb":
		request := utils.TryParseRequest[interfaces.TcbRequest](logger, contents)
		HandleTcb(writer, logger, state, request)
	case "initialized":
		{
		}
	default:
		if utils.MostRecentId == 0 || method == "" {
			break
		}

		log.Println("Not handling request for:", method)
	}
}
