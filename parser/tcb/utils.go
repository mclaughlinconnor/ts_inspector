package tcb

import (
	"fmt"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func BuildTcbBlock(state *parser.State, file *parser.File) (*Statement, error) {
	classes := file.Snapshot().Classes
	if len(classes) == 0 {
		return nil, fmt.Errorf("no resolved classes on file %v", file.Filename())
	}

	class := classes[0]
	content := []byte(file.Snapshot().Content)

	root, err := utils.ParseText(content, utils.Pug)
	if err != nil {
		return nil, err
	}

	tcb, err := GenerateTcb(state, class, root, content)
	if err != nil {
		return nil, err
	}

	return tcb, nil
}

func PugToTsLocation(state *parser.State, file *parser.File, start int, end int) (*Part, error) {
	tcb, err := BuildTcbBlock(state, file)
	if err != nil {
		return nil, err
	}

	return tcb.PugToTsLocation(start, end), nil
}

func TsToPugLocation(state *parser.State, file *parser.File, start int, end int) (*Part, error) {
	tcb, err := BuildTcbBlock(state, file)
	if err != nil {
		return nil, err
	}

	return tcb.TsToPugLocation(start, end), nil
}
