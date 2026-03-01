package interfaces

import (
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type TextDocumentIdentifier struct {
	Uri string `json:"uri"`
}

type TextDocumentItem struct {
	// The text document's URI.
	Uri string `json:"uri"`

	// The text document's language identifier.
	LanguageId string `json:"languageId"`

	// The version number of this document (it will increase after each change, including undo/redo).
	Version int `json:"version"`

	// The content of the opened text document.
	Text string `json:"text"`
}

type Location struct {
	Uri   string      `json:"uri"`
	Range utils.Range `json:"range"`
}

func NodeToLocation(node *sitter.Node, Uri string) Location {
	start := node.StartPoint()
	end := node.EndPoint()

	startPosition := utils.PositionFromPoint(start)
	endPosition := utils.PositionFromPoint(end)

	return Location{Uri: Uri, Range: utils.Range{End: endPosition, Start: startPosition}}
}

func NodeToLocationWithOffsetNode(node *sitter.Node, offsetNode *sitter.Node, content string, Uri string) Location {
	startByte := node.StartByte()
	endByte := node.EndByte()

	startByte += offsetNode.StartByte()
	endByte += offsetNode.StartByte()

	startPosition := utils.GetPositionForOffset(content, startByte)
	endPosition := utils.GetPositionForOffset(content, endByte)

	return Location{Uri: Uri, Range: utils.Range{End: endPosition, Start: startPosition}}
}

func OffsetNodeByNode(node *sitter.Node, offsetNode *sitter.Node) (uint32, uint32) {
	startByte := node.StartByte()
	endByte := node.EndByte()

	startByte += offsetNode.StartByte()
	endByte += offsetNode.StartByte()

	return startByte, endByte
}
