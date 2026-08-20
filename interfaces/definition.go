package interfaces

import "ts_inspector/utils"

type DefinitionParams struct {
	Position utils.Position

	TextDocument TextDocumentIdentifier
}

type DefinitionRequest struct {
	RequestMessage
	Params DefinitionParams `json:"params"`
}

type DefinitionResponse struct {
	ResponseMessage
	Result []Location `json:"result"`
}
