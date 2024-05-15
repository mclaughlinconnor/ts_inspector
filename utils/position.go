package utils

import sitter "github.com/smacker/go-tree-sitter"

type Position struct {
	Line uint32 `json:"line"`

	Character uint32 `json:"character"`
}

func PositionFromPoint(point sitter.Point) Position {
	return Position{Line: point.Row, Character: point.Column}
}
