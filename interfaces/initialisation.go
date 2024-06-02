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
	CodeActionProvider     bool                 `json:"codeActionProvider"` // angular uses CodeActionOptions, but I don't support that yet
	CompletionProvider     CompletionOptions    `json:"completionProvider"`
	TextDocumentSync       int                  `json:"textDocumentSync"`
	CodeLensProvider       CodeLensOptions      `json:"codeLensProvider"`
	DefinitionProvider     bool                 `json:"definitionProvider"`
	FoldingRangeProvider   bool                 `json:"foldingRangeProvider"`
	HoverProvider          bool                 `json:"hoverProvider"`
	ReferencesProvider     bool                 `json:"referencesProvider"`
	RenameOptions          RenameOptions        `json:"renameOptions"`
	SignatureHelpProvider  SignatureHelpOptions `json:"signatureHelpProvider"`
	TypeDefinitionProvider bool                 `json:"typeDefinitionProvider"`
	Workspace              struct {
		WorkspaceFolders struct {
			Supported bool `json:"supported"`
		} `json:"workspaceFolders"`
	} `json:"workspace"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

type RenameOptions struct {
	PrepareProvider bool `json:"prepareProvider"`
}

type SignatureHelpOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters"`
	RetriggerCharacters []string `json:"retriggerCharacters"`
}
