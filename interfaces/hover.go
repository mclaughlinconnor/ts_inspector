package interfaces

import "ts_inspector/utils"

type markupKind struct {
	PlainText string
	Markdown  string
}

var MarkupKind = markupKind{"plaintext", "markdown"}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type HoverParams struct {
	Position     utils.Position
	TextDocument TextDocumentIdentifier
}

type HoverRequest struct {
	Request
	Params HoverParams `json:"params"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *utils.Range  `json:"range"`
}

type HoverResponse struct {
	Response
	Result Hover `json:"result"`
}
