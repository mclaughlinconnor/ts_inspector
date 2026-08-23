package parser

import (
	"net/url"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/utils"
)

func tsgoHandleFileExistsResponse(tsgo *TsGo, request FileExistsRequest, found *bool, err error) {
	if err != nil {
		tsgo.logger.Println(err)
	}

	response := FileExistsResponse{
		TsGoResponse: TsGoResponse{RPC: "2.0", ID: request.ID},
		Result:       found,
	}

	utils.WriteResponse(tsgo.stdin, response)
}

func tsgoHandleFileExists(tsgo *TsGo, request FileExistsRequest) {
	path := request.Params

	if !strings.HasSuffix(path, interfaces.TCB_FILENAME_SUFFIX) {
		tsgoHandleFileExistsResponse(tsgo, request, nil, nil)
		return
	}

	parsedUrl, err := url.Parse(path)
	if err != nil {
		tsgoHandleFileExistsResponse(tsgo, request, nil, err)
		return
	}

	parsedUrl.Path = strings.TrimSuffix(parsedUrl.Path, interfaces.TCB_FILENAME_SUFFIX) + ".pug"
	fileUrl := parsedUrl.String()

	_, found := tsgo.state.GetFile(FilenameFromUri(fileUrl))
	tsgoHandleFileExistsResponse(tsgo, request, &found, nil)
}
