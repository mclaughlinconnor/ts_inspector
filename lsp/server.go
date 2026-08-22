package lsp

import (
	"bufio"
	"log"
	"os"
	"runtime/debug"

	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/rpc"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

var Shutdown = make(chan int, 1)

var lspReady = true
var lspIdHandler map[int]func(*utils.Writer, *log.Logger, []byte) = map[int]func(*utils.Writer, *log.Logger, []byte){}
var lspPendingMessages [][]byte = [][]byte{}

func Start() {
	state, err := parser.CreateState()
	if err != nil {
		panic(err)
	}

	state.SetTcbGenerator(func(s *parser.State, c *parser.Class, r *sitter.Node, co []byte) (string, error) {
		tcbBlock, err := tcb.GenerateTcb(s, c, r, co)
		if err != nil {
			return "", err
		}

		return tcbBlock.ToString(), nil
	})

	logger := state.Logger

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)
	big := 1024 * 1024 // 1 mb
	buf := make([]byte, big)
	scanner.Buffer(buf, big)

	writer := utils.NewWriter(os.Stdout)

	handleBytes := func(msg []byte) {
		method, contents, err := rpc.DecodeMessage(msg)
		logger.Println(method)
		if err != nil {
			logger.Printf("Error: %s", err)
			return
		}

		handleMessage(logger, writer, &state, method, contents)
	}

	for scanner.Scan() {
		logger.Println("Scanner found the next message")
		msg := scanner.Bytes()
		logger.Println("Received msg", string(msg))

		if !lspReady {
			lspPendingMessages = append(lspPendingMessages, msg)
			continue
		}

		if lspReady && len(lspPendingMessages) != 0 {
			for _, m := range lspPendingMessages {
				handleBytes(m)
			}

			lspPendingMessages = [][]byte{}
		}

		handleBytes(msg)
	}

	logger.Println("LSP event loop finished")

	if err := scanner.Err(); err != nil {
		logger.Fatal(err)
	}
}

func handleMessage(logger *log.Logger, writer *utils.Writer, state *parser.State, method string, contents []byte) {
	defer func() {
		if r := recover(); r != nil {
			logger.Println("Panicked with: ", r, "responding with empty response")
			logger.Println("Stack: ", string(debug.Stack()))
		}
	}()

	r := utils.TryParseRequest[interfaces.RequestMessage](logger, contents)

	logger.Printf("Received msg with method: %s", method)

	switch method {
	case "initialize":
		request := utils.TryParseRequest[interfaces.InitializeRequest](logger, contents)
		go lspHandleInitialise(writer, logger, state, request)
	case "shutdown":
		Shutdown <- 1
	case "textDocument/didOpen":
		request := utils.TryParseRequest[interfaces.DidOpenTextDocumentNotification](logger, contents)
		lspHandleDidOpen(writer, logger, state, request)
	case "textDocument/didChange":
		request := utils.TryParseRequest[interfaces.DidChangeTextDocumentNotification](logger, contents)
		lspHandleDidChange(writer, logger, state, request)
	case "textDocument/codeAction":
		request := utils.TryParseRequest[interfaces.CodeActionRequest](logger, contents)
		lspHandleCodeAction(writer, logger, state, request)
	case "textDocument/completion":
		request := utils.TryParseRequest[interfaces.CompletionRequest](logger, contents)
		lspHandleCompletion(writer, logger, state, request)
	case "textDocument/definition":
		request := utils.TryParseRequest[interfaces.DefinitionRequest](logger, contents)
		lspHandleDefinition(writer, logger, state, request)
	case "textDocument/hover":
		request := utils.TryParseRequest[interfaces.HoverRequest](logger, contents)
		lspHandleHover(writer, logger, state, request)
	case "textDocument/references":
		request := utils.TryParseRequest[interfaces.ReferenceRequest](logger, contents)
		lspHandleReferences(writer, logger, state, request)
	case "workspace/executeCommand":
		request := utils.TryParseRequest[interfaces.ExecuteCommandRequest](logger, contents)
		lspHandleExecuteCommand(writer, logger, state, request)
	case "workspace/symbol":
		request := utils.TryParseRequest[interfaces.WorkspaceSymbolRequest](logger, contents)
		lspHandleWorkspaceSymbol(writer, logger, state, request)
	case "ts_inspector/getTcb":
		request := utils.TryParseRequest[interfaces.TcbRequest](logger, contents)
		lspHandleTcb(writer, logger, state, request)
	case "initialized":
		{
			lspReady = false
		}
	default:
		handler, found := lspIdHandler[r.ID]
		if handler == nil || !found {
			log.Println("Not handling request for:", method)
			break
		}

		handler(writer, logger, contents)
	}
}

func emptyResponse(writer *utils.Writer, requestId int) {
	utils.WriteResponse(writer, interfaces.EmptyResponse{Result: nil, ResponseMessage: interfaces.ResponseMessage{ID: &requestId, RPC: "2.0"}})
}

func logError(writer *utils.Writer, logger *log.Logger, err error) {
	notification := interfaces.BuildMessageNotification(err.Error(), interfaces.MessageType.Error)
	utils.WriteResponse(writer, notification)

	logger.Println(err)
}

func logErrorWithResponse(writer *utils.Writer, logger *log.Logger, err error, requestId int) {
	logError(writer, logger, err)
	emptyResponse(writer, requestId)
}
