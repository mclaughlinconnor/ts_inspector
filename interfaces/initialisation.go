package interfaces

type InitializeRequest struct {
	Request
}

type InitializeResponse struct {
	Response
	Result InitializeResult `json:"result"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type ServerCapabilities struct {
	CodeActionProvider bool              `json:"codeActionProvider"`
	CompletionProvider CompletionOptions `json:"completionProvider"`
	TextDocumentSync   int               `json:"textDocumentSync"`
}
