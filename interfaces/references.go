package interfaces

import "ts_inspector/utils"

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type ReferenceParams struct {
	Context      ReferenceContext
	Position     utils.Position
	TextDocument TextDocumentIdentifier
}

type ReferenceRequest struct {
	RequestMessage
	Params ReferenceParams `json:"params"`
}

type ReferenceResponse struct {
	ResponseMessage
	Result []Location `json:"result"`
}
