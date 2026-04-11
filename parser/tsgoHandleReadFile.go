package parser

import (
	"net/url"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func tsgoHandleReadFileResponse(tsgo *TsGo, request ReadFileRequest, content *Content) {
	response := ReadFileResponse{
		TsGoResponse: TsGoResponse{RPC: "2.0", ID: request.ID},
		Result:       content,
	}

	utils.WriteResponse(*tsgo.stdin, response)
}

func tsgoHandleReadFile(tsgo *TsGo, request ReadFileRequest) {
	path := request.Params

	if !strings.HasSuffix(path, interfaces.TCB_FILENAME_SUFFIX) {
		tsgoHandleReadFileResponse(tsgo, request, nil)
		return
	}

	parsedUrl, err := url.Parse(path)
	if err != nil {
		tsgoHandleReadFileResponse(tsgo, request, nil)
		return
	}

	parsedUrl.Path = strings.TrimSuffix(parsedUrl.Path, interfaces.TCB_FILENAME_SUFFIX) + ".pug"
	fileUrl := parsedUrl.String()

	file, _ := tsgo.state.GetFile(FilenameFromUri(fileUrl))
	if file == nil || file.Snapshot().Filetype != "pug" {
		tsgoHandleReadFileResponse(tsgo, request, &Content{Content: file.Snapshot().Content})
		return
	}

	generateTcb := tsgo.state.GetTcbGenerator()
	if generateTcb == nil {
		tsgoHandleReadFileResponse(tsgo, request, nil)
		return
	}

	content := []byte(file.Snapshot().Content)
	tcbBlock, err := utils.ParseText(content, utils.Pug, "", func(root *sitter.Node, _ []byte, _ string) (string, error) {
		classes := file.Snapshot().Classes
		if len(classes) == 0 {
			return "", nil
		}

		tcb := generateTcb(tsgo.state, classes[0], root, content)

		return tcb, nil
	})

	if err != nil {
		tcbBlock = err.Error()
	}

	tsgoHandleReadFileResponse(tsgo, request, &Content{Content: tcbBlock})
}
