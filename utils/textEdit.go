package utils

type TextEdit struct {
	Range Range `json:"range"`

	NewText string `json:"newText"`
}

type TextEdits = []TextEdit
