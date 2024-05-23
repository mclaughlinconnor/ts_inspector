package lsp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
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
	state := parser.State{}

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Error: %s", err)
			continue
		}

		state = handleMessage(logger, writer, state, method, contents)
	}
}

func handleMessage(logger *log.Logger, writer io.Writer, state parser.State, method string, contents []byte) parser.State {
	logger.Printf("Received msg with method: %s", method)

	switch method {
	case "initialize":
		request := TryParseRequest[InitializeRequest](logger, contents)
		HandleInitialise(writer, logger, request)
	case "textDocument/didOpen":
		request := TryParseRequest[DidOpenTextDocumentNotification](logger, contents)
		state = HandleDidOpen(writer, logger, state, request)
	case "textDocument/didChange":
		request := TryParseRequest[DidChangeTextDocumentNotification](logger, contents)
		state = HandleDidChange(writer, logger, state, request)
	case "textDocument/codeAction":
		request := TryParseRequest[CodeActionRequest](logger, contents)
		HandleCodeAction(writer, logger, state, request)
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
