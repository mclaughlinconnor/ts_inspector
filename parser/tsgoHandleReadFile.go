package parser

import (
	"net/url"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/utils"
)

func tsgoHandleReadFileResponse(tsgo *TsGo, request ReadFileRequest, content *Content) {
	response := ReadFileResponse{
		TsGoResponse: TsGoResponse{RPC: "2.0", ID: request.ID},
		Result:       content,
	}

	utils.WriteResponse(tsgo.stdin, response)
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

	file, found := tsgo.state.GetFile(FilenameFromUri(fileUrl))
	if !found {
		tsgoHandleReadFileResponse(tsgo, request, &Content{Content: ""})
		return
	}

	if file.Snapshot().Filetype != "pug" {
		tsgoHandleReadFileResponse(tsgo, request, &Content{Content: file.Snapshot().Content})
		return
	}

	generateTcb := tsgo.state.GetTcbGenerator()
	if generateTcb == nil {
		tsgoHandleReadFileResponse(tsgo, request, nil)
		return
	}

	content := []byte(file.Snapshot().Content)
	root, err := utils.ParseText(content, utils.Pug)
	if err != nil {
		tsgoHandleReadFileResponse(tsgo, request, &Content{Content: err.Error()})
		return
	}

	classes := file.Snapshot().Classes
	if len(classes) == 0 {
		tsgoHandleReadFileResponse(tsgo, request, nil)
		return
	}

	tcbBlock, err := generateTcb(tsgo.state, classes[0], root, content)
	if err != nil {
		tsgo.logger.Println(err)
		tsgoHandleReadFileResponse(tsgo, request, &Content{Content: ""})

		return
	}

	tsgoHandleReadFileResponse(tsgo, request, &Content{Content: tcbBlock})
}
