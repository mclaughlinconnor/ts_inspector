package lsp

import sitter "github.com/smacker/go-tree-sitter"

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

type Range struct {
	Start Position `json:"start"`

	End Position `json:"end"`
}

type Position struct {
	Line uint32

	Character uint32
}

func PositionFromPoint(point sitter.Point) Position {
	return Position{point.Row, point.Column}
}
