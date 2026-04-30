package tcb

import (
	"fmt"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func BuildTcbBlock(state *parser.State, file *parser.File) (*Statement, error) {
	classes := file.Snapshot().Classes
	if len(classes) == 0 {
		return nil, fmt.Errorf("No resolved classes on file %v", file.Filename())
	}

	class := classes[0]
	content := []byte(file.Snapshot().Content)

	return utils.ParseText(content, utils.Pug, nil, func(root *sitter.Node, _ []byte, _ *Statement) (*Statement, error) {
		tcb := GenerateTcb(state, class, root, content)

		return tcb, nil
	})
}

func PugToTsLocation(state *parser.State, file *parser.File, start int, end int) *Part {
	tcb, err := BuildTcbBlock(state, file)
	if err != nil {
		return nil
	}

	return tcb.PugToTsLocation(start, end)
}

func TsToPugLocation(state *parser.State, file *parser.File, start int, end int) *Part {
	tcb, err := BuildTcbBlock(state, file)
	if err != nil {
		return nil
	}

	return tcb.TsToPugLocation(start, end)
}
