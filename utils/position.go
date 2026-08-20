package utils

import (
	sitter "github.com/smacker/go-tree-sitter"
)

type Position struct {
	Line uint32 `json:"line"`

	Character uint32 `json:"character"`
}

func PositionFromPoint(point sitter.Point) Position {
	return Position{Line: point.Row, Character: point.Column}
}

func GetPositionForOffset(content string, offset uint32) Position {
	lineOffsets := GetLineOffsets(content)

	if offset >= uint32(len(content)) {
		return Position{Line: uint32(len(lineOffsets)) - 1, Character: 0}
	} else if offset < 0 {
		return Position{Line: 0, Character: 0}
	}

	var line uint32
	var character uint32

	for index, lineOffset := range lineOffsets {
		if lineOffset > offset {
			if index > 0 {
				line = uint32(index - 1)
				character = offset - lineOffsets[index-1]
			} else {
				line = 0
				character = offset
			}

			break
		}
	}

	return Position{Line: line, Character: character}
}

func GetLineOffsets(text string) []uint32 {
	var i uint32 = 0

	offsets := []uint32{}
	isLineStart := true

	textLength := uint32(len(text))
	for i < textLength {
		if isLineStart {
			offsets = append(offsets, i)
			isLineStart = false
		}

		ch := text[i]
		isLineStart = ch == '\r' || ch == '\n'

		if ch == '\r' && i+1 < textLength && text[i+1] == '\n' {
			i++
		}

		i++
	}

	if isLineStart && textLength > 0 {
		offsets = append(offsets, textLength)
	}

	return offsets
}

func ZeroPosition() Position {
	return Position{0, 0}
}
