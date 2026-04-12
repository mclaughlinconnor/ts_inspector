package parser

import (
	"net/url"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/utils"
)

func tsgoHandleFileExistsResponse(tsgo *TsGoApi, request FileExistsRequest, found *bool) {
	response := FileExistsResponse{
		TsGoResponse: TsGoResponse{RPC: "2.0", ID: request.ID},
		Result:       found,
	}

	utils.WriteResponse(*tsgo.connection, response)
}

func tsgoHandleFileExists(tsgo *TsGoApi, request FileExistsRequest) {
	path := request.Params

	if !strings.HasSuffix(path, interfaces.TCB_FILENAME_SUFFIX) {
		tsgoHandleFileExistsResponse(tsgo, request, nil)
		return
	}

	parsedUrl, err := url.Parse(path)
	if err != nil {
		tsgoHandleFileExistsResponse(tsgo, request, nil)
		return
	}

	parsedUrl.Path = strings.TrimSuffix(parsedUrl.Path, interfaces.TCB_FILENAME_SUFFIX) + ".pug"
	fileUrl := parsedUrl.String()

	_, found := tsgo.state.GetFile(FilenameFromUri(fileUrl))
	tsgoHandleFileExistsResponse(tsgo, request, &found)
}
