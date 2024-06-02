package interfaces

type textDocumentSyncKind struct {
	None        int
	Full        int
	Incremental int
}

var TextDocumentSyncKind = textDocumentSyncKind{0, 1, 2}

type DidOpenTextDocumentNotification struct {
	Notification
	Params DidOpenTextDocumentNotificationParams `json:"params"`
}

type DidOpenTextDocumentNotificationParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}
