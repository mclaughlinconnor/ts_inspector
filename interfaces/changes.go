package interfaces

type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type DidChangeTextDocumentNotification struct {
	Notification
	Params DidChangeTextDocumentNotificationParams `json:"params"`
}

type DidChangeTextDocumentNotificationParams struct {
	TextDocument   TextDocumentItem                 `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}
