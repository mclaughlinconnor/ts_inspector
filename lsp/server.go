package lsp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"ts_inspector/actions"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/ngserver"
	"ts_inspector/parser"
	"ts_inspector/pug"
	"ts_inspector/rpc"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

var Shutdown = make(chan int, 1)

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
	defer func() {
		if r := recover(); r != nil {
			logger.Println("Panicked with: ", r)
			return
		}
	}()

	logger.Printf("Received msg with method: %s", method)

	switch method {
	case "initialize":
		ngserver.SendToAngular(string(msg))
		request := utils.TryParseRequest[interfaces.InitializeRequest](logger, contents)
		ngserver.Requests[request.ID] = ngserver.RequestData{Method: method}
	case "shutdown":
		Shutdown <- 1
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
		ngserver.Requests[request.ID] = ngserver.RequestData{Method: method}
		HandleCodeAction(writer, logger, state, request)
	case "completionItem/resolve":
		request := utils.TryParseRequest[interfaces.CompletionItemRequest](logger, contents)
		ngserver.Requests[request.ID] = ngserver.RequestData{Method: method}

		file, found := state[request.Params.Data["filePath"].(string)]
		if !found {
			// should be found, but if there's a panic in open handling, this can cause infinite recursion
			return state
		}
		_, _ = utils.GetRootNode(false, file.Content, utils.Pug)

		parseResult, err := pug.Parse(file.Content)
		if err != nil {
			return state
		}

		pugOffset := uint32(request.Params.Data["CM_Position"].(float64))
		htmlPosition := pug.PugLocationToHtmlLocation(pugOffset, parseResult) + 1
		htmlOffset := parser.GetPositionForOffset(parseResult.HtmlText, htmlPosition)

		request.Params.Data["position"] = htmlOffset

		// In completionresolve, use this position if it exists

		updatedMsg := rpc.EncodeMessage(request)
		ngserver.SendToAngular(string(updatedMsg))
	case "textDocument/completion":
		request := utils.TryParseRequest[interfaces.CompletionRequest](logger, contents)

		file, found := state[parser.FilenameFromUri(request.Params.TextDocument.Uri)]
		if !found {
			// should be found, but if there's a panic in open handling, this can cause infinite recursion
			return state
		}
		root, err := utils.GetRootNode(false, file.Content, utils.Pug)

		offset := file.GetOffsetForPosition(request.Params.Position)
		parseResult, err := pug.Parse(file.Content)
		if err != nil {
			return state
		}

		pugOffset := file.GetOffsetForPosition(request.Params.Position)
		htmlPosition := pug.PugLocationToHtmlLocation(pugOffset, parseResult)
		htmlOffset := parser.GetPositionForOffset(parseResult.HtmlText, htmlPosition)

		ngserver.Requests[request.ID] = ngserver.RequestData{Method: method, Position: &pugOffset}

		quotedAttributeNode := ast.HasNodeInHierarchy(root, "quoted_attribute_value", offset, offset)
		isInQuotedAttribute := quotedAttributeNode != nil
		contentNode := ast.HasNodeInHierarchy(root, "content", offset, offset)
		isInContent := contentNode != nil
		attributesNode := ast.HasNodeInHierarchy(root, "attributes", offset, offset)
		isInAttributes := attributesNode != nil

		if isInQuotedAttribute {
			request.Params.Position = htmlOffset
			request.Method = "cm/getPropertyExpressionCompletion"
		} else if isInContent {
			isInInterpolation, _ := utils.ParseText([]byte(contentNode.Content([]byte(file.Content))), utils.AngularContent, false, func(root *sitter.Node, content []byte, isInInterpolation bool) (bool, error) {
				angularOffset := offset - contentNode.StartByte()
				return ast.HasNodeInHierarchy(root, "interpolation", angularOffset, angularOffset) != nil, nil
			})

			if !isInInterpolation {
				return state
			}

			request.Params.Position = htmlOffset
			request.Method = "cm/getPropertyExpressionCompletion"
		} else if isInAttributes {
			request.Params.Position = htmlOffset
			request.Method = "cm/getAttrCompletion"
		} else {
			request.Method = "cm/getTagCompletion"
		}

		updatedMsg := rpc.EncodeMessage(request)
		ngserver.SendToAngular(string(updatedMsg))
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
