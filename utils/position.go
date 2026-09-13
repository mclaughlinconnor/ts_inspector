package utils

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Position struct {
	Line uint `json:"line"`

	Character uint `json:"character"`
}

func LspPositionFromTsPosition(point sitter.Point) Position {
	return Position{Line: point.Row, Character: point.Column}
}

func GetPositionForOffset(content string, offset uint) Position {
	lineOffsets := GetLineOffsets(content)

	if offset >= uint(len(content)) {
		return Position{Line: uint(len(lineOffsets)) - 1, Character: 0}
	}

	var line uint
	var character uint

	for index, lineOffset := range lineOffsets {
		if lineOffset > offset {
			if index > 0 {
				line = uint(index - 1)
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

func GetPositionForOffset2(content string, offset int) Position {
	lineOffsets := GetLineOffsets2(content)

	if offset >= len(content) {
		return Position{Line: uint(len(lineOffsets)) - 1, Character: 0}
	}

	var line int
	var character int

	if len(lineOffsets) == 1 {
		return Position{Line: 0, Character: uint(offset)}
	}

	for index, lineOffset := range lineOffsets {
		if lineOffset > offset {
			if index > 0 {
				line = index - 1
				character = offset - lineOffsets[index-1]
			} else {
				line = 0
				character = offset
			}

			break
		}
	}

	return Position{Line: uint(line), Character: uint(character)}
}

func GetLineOffsets(text string) []uint {
	var i uint = 0

	offsets := []uint{}
	isLineStart := true

	textLength := uint(len(text))
	for i < textLength {
		if isLineStart {
			offsets = append(offsets, i)
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

func GetLineOffsets2(text string) []int {
	var i = 0

	offsets := []int{}
	isLineStart := true

	textLength := int(len(text))
	for i < textLength {
		if isLineStart {
			offsets = append(offsets, i)
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
