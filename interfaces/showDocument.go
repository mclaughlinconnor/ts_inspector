package interfaces

import "ts_inspector/utils"

type ShowDocumentParams struct {
	Uri       string       `json:"uri"`
	External  *bool        `json:"external,omitempty"`
	TakeFocus *bool        `json:"takeFocus,omitempty"`
	Selection *utils.Range `json:"selection,omitempty"`
}

type ShowDocumentNotification struct {
	Notification
	Params ShowDocumentParams `json:"params"`
}
