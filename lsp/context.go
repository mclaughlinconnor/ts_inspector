package lsp

import (
	"fmt"
	"log"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
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
	cursorPosition       utils.Position
	cursorRange          utils.Range
	isOnAttrName         bool
	isOnTagName          bool
	namedNodeUnderCursor *sitter.Node
	rootNode             *sitter.Node
	tagUnderCursor       *ast.Tag
}

func buildContext(writer *utils.Writer, logger *log.Logger, state *parser.State, textDocument interfaces.TextDocumentIdentifier, position utils.Position) (*context, error) {
	file, _ := state.GetFile(parser.FilenameFromUri(textDocument.Uri))
	if file == nil {
		if strings.HasSuffix(textDocument.Uri, interfaces.TCB_FILENAME_SUFFIX) {
			return nil, nil
		}

		return nil, fmt.Errorf("filenot found %v", textDocument.Uri)
	}

	cursorOffset := file.GetOffsetForPosition(position)
	cursorPosition := position
	cursorRange := utils.RangeFromStart(cursorPosition)

	content := []byte(file.Snapshot().Content)

	rootNode := utils.ParseText(content, utils.Pug)

	attributeUnderCursor, isOnAttrName := ast.GetAttributeNameAtOffset2(rootNode, file.Snapshot().Content, cursorOffset)
	tagUnderCursor, isOnTagName := ast.GetTagAtOffset2(rootNode, file.Snapshot().Content, cursorOffset)

	rootNode = utils.ParseText(content, utils.Pug)

	namedNodeUnderCursor := ast.GetNamedNodeAtPosition(rootNode, cursorOffset)

	ci := cursorInfo{
		attributeUnderCursor: attributeUnderCursor,
		cursorOffset:         int(cursorOffset),
		cursorPosition:       cursorPosition,
		cursorRange:          cursorRange,
		isOnAttrName:         isOnAttrName,
		isOnTagName:          isOnTagName,
		namedNodeUnderCursor: namedNodeUnderCursor,
		rootNode:             rootNode,
		tagUnderCursor:       tagUnderCursor,
	}

	return &context{
		file:   file,
		ci:     &ci,
		logger: logger,
		state:  state,
		writer: writer,
	}, nil
}
