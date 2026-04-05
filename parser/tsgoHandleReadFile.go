package parser

import (
	"net/url"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func tsgoHandleReadFile(state *State, request ReadFileRequest) *Content {
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

	file, _ := state.GetFile(FilenameFromUri(fileUrl))
	if file == nil || file.Snapshot().Filetype != "pug" {
		return &Content{Content: file.Snapshot().Content}
	}

	generateTcb := state.GetTcbGenerator()
	if generateTcb == nil {
		return nil
	}

	content := []byte(file.Snapshot().Content)
	tcbBlock, err := utils.ParseText(content, utils.Pug, "", func(root *sitter.Node, _ []byte, _ string) (string, error) {
		tcb := generateTcb(state, file.Snapshot().Classes[0], root, content)

		return tcb, nil
	})

	if err != nil {
		tcbBlock = err.Error()
	}

	return &Content{Content: tcbBlock}
}
