package interfaces

import "ts_inspector/utils"

type DefinitionParams struct {
	Position utils.Position

	TextDocument TextDocumentIdentifier
}

type DefinitionRequest struct {
	Request[int]
	Params DefinitionParams `json:"params"`
}

type DefinitionResponse struct {
	Response
	Result []Location `json:"result"`
}
