package lsp

import (
	"bufio"
	"io"
	"log"
	"os"
	"runtime/debug"

	"ts_inspector/actions"
	"ts_inspector/ast"
	"ts_inspector/commands"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/pug"
	"ts_inspector/rpc"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

var Shutdown = make(chan int, 1)

func Start() {
	logger := utils.GetLogger("ts_inspector")
	logger.Println("Started")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	writer := os.Stdout

	utils.InitQueries()
	actions.InitActions()
	commands.InitCommands()
	state := parser.State{Files: map[string]parser.File{}}

	for scanner.Scan() {
		msg := scanner.Bytes()
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
}

var lastCompletionId int

func handleMessage(logger *log.Logger, writer io.Writer, state parser.State, method string, contents []byte, msg []byte) (parser.State, bool) {
	defer func() (parser.State, bool) {
		if r := recover(); r != nil {
			logger.Println("Panicked with: ", r, "responding with empty response")
			logger.Println("Stack: ", string(debug.Stack()))
			utils.WriteResponse(writer, Response{RPC: "2.0", ID: &utils.MostRecentId})

		}

		return state, false
	}()

	r := utils.TryParseRequest[interfaces.InitializeRequest](logger, contents)
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
	case "completionItem/resolve":
		request := utils.TryParseRequest[interfaces.CompletionItemRequest](logger, contents)

		file, found := state.Files[request.Params.Data["filePath"].(string)]
		if !found || file.Filetype != "pug" {
			response := interfaces.CompletionResponse{
				Response: interfaces.Response{RPC: "2.0", ID: &request.ID},
				Result:   []interfaces.CompletionItem{},
			}
			logger.Println("Is not pug file or is not found, nothing I can do", request.ID)
			utils.WriteResponse(writer, response)
			return state, true
		}

		_, _ = utils.GetRootNode(false, file.Content, utils.Pug)

		parseResult, err := pug.Parse(file.Content)
		if err != nil {
			return state, true
		}

		pugOffset := uint32(request.Params.Data["CM_Position"].(float64))
		htmlPosition := pug.PugLocationToHtmlLocation(pugOffset, parseResult) + 1
		htmlOffset := parser.GetPositionForOffset(parseResult.HtmlText, htmlPosition)

		request.Params.Data["position"] = htmlOffset
		// Do nothing
	case "textDocument/completion":
		request := utils.TryParseRequest[interfaces.CompletionRequest](logger, contents)

		file, found := state.Files[parser.FilenameFromUri(request.Params.TextDocument.Uri)]
		if !found || file.Filetype != "pug" {
			response := interfaces.CompletionResponse{
				Response: interfaces.Response{RPC: "2.0", ID: &request.ID},
				Result:   []interfaces.CompletionItem{},
			}
			logger.Println("Is not pug file or is not found, nothing I can do", request.ID)
			utils.WriteResponse(writer, response)
			return state, true
		}

		root, err := utils.GetRootNode(false, file.Content, utils.Pug)

		offset := file.GetOffsetForPosition(request.Params.Position)
		parseResult, err := pug.Parse(file.Content)
		if err != nil {
			return state, true
		}

		pugOffset := file.GetOffsetForPosition(request.Params.Position)
		htmlPosition := pug.PugLocationToHtmlLocation(pugOffset, parseResult)
		htmlOffset := parser.GetPositionForOffset(parseResult.HtmlText, htmlPosition)

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
				return state, true
			}

			request.Params.Position = htmlOffset
			request.Method = "cm/getPropertyExpressionCompletion"
		} else if isInAttributes {
			request.Params.Position = htmlOffset
			request.Method = "cm/getAttrCompletion"
		} else {
			request.Method = "cm/getTagCompletion"
		}

		// Do nothing
	}

	return state, true
}
