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
	Request[int]
	Params ReferenceParams `json:"params"`
}

type ReferenceResponse struct {
	Response
	Result []Location `json:"result"`
}
