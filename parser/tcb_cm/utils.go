package tcb_cm

import (
	"fmt"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func BuildTcbBlock(state *parser.State, file *parser.File) (*StatementParts, error) {
	classes := file.Snapshot().Classes
	if len(classes) == 0 {
		return nil, fmt.Errorf("No resolved classes on file %v", file.Filename())
	}

	class := classes[0]
	content := []byte(file.Snapshot().Content)

	return utils.ParseText(content, utils.Pug, nil, func(root *sitter.Node, _ []byte, _ *StatementParts) (*StatementParts, error) {
		tcb := GenerateTcb(state, class, root, content)

		return tcb, nil
	})
}
