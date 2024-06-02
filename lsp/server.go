package lsp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"ts_inspector/actions"
	"ts_inspector/interfaces"
	"ts_inspector/ngserver"
	"ts_inspector/parser"
	"ts_inspector/rpc"
	"ts_inspector/utils"
)

func Start() {
	logger := getLogger("/home/connor/Development/ts_inspector/log.txt")
	logger.Println("Started")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	writer := os.Stdout

	utils.InitQueries()
	actions.InitActions()
	state := parser.State{}

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		logger.Println(method)
		if err != nil {
			logger.Printf("Error: %s", err)
			continue
		}

		state = handleMessage(logger, writer, state, method, contents, msg)
	}
}

var lastCompletionId int

func handleMessage(logger *log.Logger, writer io.Writer, state parser.State, method string, contents []byte, msg []byte) parser.State {
	logger.Printf("Received msg with method: %s", method)

	switch method {
	case "initialize":
		ngserver.SendToAngular(string(msg))
		request := utils.TryParseRequest[interfaces.InitializeRequest](logger, contents)
		ngserver.Requests[request.ID] = method
	case "textDocument/didOpen":
		request := utils.TryParseRequest[interfaces.DidOpenTextDocumentNotification](logger, contents)
		state = HandleDidOpen(writer, logger, state, request)
		ngserver.SendToAngular(string(msg))
	case "textDocument/didChange":
		request := utils.TryParseRequest[interfaces.DidChangeTextDocumentNotification](logger, contents)
		state = HandleDidChange(writer, logger, state, request)
		ngserver.SendToAngular(string(msg))
	case "textDocument/codeAction":
		request := utils.TryParseRequest[interfaces.CodeActionRequest](logger, contents)
		ngserver.Requests[request.ID] = method
		HandleCodeAction(writer, logger, state, request)
	case "textDocument/completion":
		request := utils.TryParseRequest[interfaces.CompletionRequest](logger, contents)
		ngserver.Requests[request.ID] = method
		ngserver.SendToAngular(string(msg))
	default:
		ngserver.SendToAngular(string(msg))
	}

	return state
}

func getLogger(filename string) *log.Logger {
	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("Invalid file: %s", filename))
	}

	return log.New(logfile, "[ts_inspector]", log.Ldate|log.Ltime|log.Lshortfile)
}
