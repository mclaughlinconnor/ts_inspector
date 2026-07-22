package parser

import (
	"os"
	"path"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/utils"
)

func tsgoHandleGetAccessibleEntriesResponse(t *TsGo, request GetAccessibleEntriesRequest, entries *Entries) {
	response := GetAcceessibleEntriesResponse{
		TsGoResponse: TsGoResponse{RPC: "2.0", ID: request.ID},
		Result:       entries,
	}

	utils.WriteResponse(*t.stdin, response)
}

func tsgoHandleGetAccessibleEntries(tsgo *TsGo, request GetAccessibleEntriesRequest) {
	requestPath := request.Params

	files := []string{}
	directories := []string{}

	stateFiles := tsgo.state.GetFiles()
	for filename := range *stateFiles {
		stateDir := path.Dir(filename) // this is super slow. cache it
		if stateDir != requestPath {
			continue
		}

		if path.Ext(filename) == ".pug" {
			files = append(files, strings.TrimSuffix(path.Base(filename), ".pug")+interfaces.TCB_FILENAME_SUFFIX)
			continue
		}
	}

	if len(files) == 0 {
		tsgoHandleGetAccessibleEntriesResponse(tsgo, request, nil)
		return
	}

	diskEntries, _ := os.ReadDir(requestPath)
	for _, entry := range diskEntries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		} else {
			files = append(files, entry.Name())
		}
	}

	entries := &Entries{Directories: directories, Files: files}
	tsgoHandleGetAccessibleEntriesResponse(tsgo, request, entries)
}
