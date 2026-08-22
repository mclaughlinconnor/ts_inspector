package lsp

import (
	"log"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

type context struct {
	file   *parser.File
	ci     *cursorInfo
	logger *log.Logger
	state  *parser.State
	writer *utils.Writer
}

type cursorInfo struct {
	attributeUnderCursor string
	cursorOffset         int
	isOnAttrName         bool
	isOnTagName          bool
	tagUnderCursor       *ast.Tag
}

func buildContext(writer *utils.Writer, logger *log.Logger, state *parser.State, textDocument interfaces.TextDocumentIdentifier, position utils.Position) *context {
	file, _ := state.GetFile(parser.FilenameFromUri(textDocument.Uri))
	if file == nil {
		return nil
	}

	cursorOffset := file.GetOffsetForPosition(position)

	attributeUnderCursor, isOnAttrName := ast.GetAttributeNameAtOffset(file.Snapshot().Content, cursorOffset)
	tagUnderCursor, isOnTagName := ast.GetTagAtOffset2(file.Snapshot().Content, cursorOffset)

	ci := cursorInfo{
		attributeUnderCursor: attributeUnderCursor,
		cursorOffset:         int(cursorOffset),
		isOnAttrName:         isOnAttrName,
		isOnTagName:          isOnTagName,
		tagUnderCursor:       tagUnderCursor,
	}

	return &context{
		file:   file,
		ci:     &ci,
		logger: logger,
		state:  state,
		writer: writer,
	}
}
