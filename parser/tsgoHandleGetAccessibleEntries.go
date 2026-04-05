package parser

import (
	"os"
	"path"
	"strings"
	"ts_inspector/interfaces"
)

func tsgoHandleGetAccessibleEntries(state *State, request GetAccessibleEntriesRequest) *Entries {
	requestPath := request.Params

	files := []string{}
	directories := []string{}

	stateFiles := state.GetFiles()
	for filename := range *stateFiles {
		stateDir := path.Dir(filename)
		if stateDir != requestPath {
			continue
		}

		if path.Ext(filename) == ".pug" {
			files = append(files, strings.TrimSuffix(path.Base(filename), ".pug")+interfaces.TCB_FILENAME_SUFFIX)
			continue
		}
	}

	if len(files) == 0 {
		return nil
	}

	diskEntries, _ := os.ReadDir(requestPath)
	for _, entry := range diskEntries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		} else {
			files = append(files, entry.Name())
		}
	}

	return &Entries{Directories: directories, Files: files}
}
