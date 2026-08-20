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
	RequestMessage
	Params HoverParams `json:"params"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *utils.Range  `json:"range,omitempty"`
}

type HoverResponse struct {
	ResponseMessage
	Result Hover `json:"result"`
}
