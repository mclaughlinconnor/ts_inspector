package manipulation

import (
	"fmt"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type astError struct {
	endOffset   uint
	message     string
	startOffset uint
	state       nodeInterface
}

func (a astError) Error() string {
	filename := a.state.getProgramFilename()

	position := utils.GetPositionForOffset(a.state.getProgramText(), a.startOffset)
	line := position.Line
	column := position.Character

	return fmt.Sprintf("%v:%v:%v: Invalid ast: %v", filename, line, column, a.message)
}

func (a astError) GetEndColumn() uint {
	position := utils.GetPositionForOffset(a.state.getProgramText(), a.endOffset)
	return position.Character
}

func (a astError) GetEndLine() uint {
	position := utils.GetPositionForOffset(a.state.getProgramText(), a.endOffset)
	return position.Line
}

func (a astError) GetFilename() string {
	return a.state.getProgramFilename()
}

func (a astError) GetMessage() string {
	return a.message
}

func (a astError) GetRange() utils.Range {
	start := utils.Position{Line: a.GetStartLine(), Character: a.GetStartColumn()}
	end := utils.Position{Line: a.GetEndLine(), Character: a.GetEndColumn()}
	return utils.Range{Start: start, End: end}
}

func (a astError) GetStartLine() uint {
	position := utils.GetPositionForOffset(a.state.getProgramText(), a.startOffset)
	return position.Line
}

func (a astError) GetStartColumn() uint {
	position := utils.GetPositionForOffset(a.state.getProgramText(), a.startOffset)
	return position.Character
}

func newAstError(node nodeInterface, message string) astError {
	err := astError{endOffset: node.getEndOffset(), message: message, startOffset: node.getStartOffset(), state: node}
	return err
}

func newAstErrorTS(node *sitter.Node, state nodeInterface, message string) astError {
	err := astError{endOffset: node.EndByte(), message: message, startOffset: node.StartByte(), state: state}
	return err
}

// nolint:unused
func newAstErrorE(node nodeInterface, state nodeInterface, error error) astError {
	return newAstError(node, "invalid ast: "+error.Error())
}

func newAstErrorETS(node *sitter.Node, state nodeInterface, error error) astError {
	return newAstErrorTS(node, state, "invalid ast: "+error.Error())
}
