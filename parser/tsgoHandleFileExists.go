package parser

import (
	"net/url"
	"strings"
	"ts_inspector/interfaces"
)

func tsgoHandleFileExists(state *State, request FileExistsRequest) *bool {
	path := request.Params

	if !strings.HasSuffix(path, interfaces.TCB_FILENAME_SUFFIX) {
		return nil
	}

	parsedUrl, err := url.Parse(path)
	if err != nil {
		return nil
	}

	parsedUrl.Path = strings.TrimSuffix(parsedUrl.Path, interfaces.TCB_FILENAME_SUFFIX) + ".pug"
	fileUrl := parsedUrl.String()

	_, found := state.GetFile(FilenameFromUri(fileUrl))
	return &found
}
