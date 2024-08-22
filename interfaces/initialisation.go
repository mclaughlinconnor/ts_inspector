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
	CodeActionProvider     bool                  `json:"codeActionProvider"` // angular uses CodeActionOptions, but I don't support that yet
	CodeLensProvider       CodeLensOptions       `json:"codeLensProvider"`
	CompletionProvider     CompletionOptions     `json:"completionProvider"`
	DefinitionProvider     bool                  `json:"definitionProvider"`
	ExecuteCommandProvider ExecuteCommandOptions `json:"executeCommandProvider"`
	FoldingRangeProvider   bool                  `json:"foldingRangeProvider"`
	HoverProvider          bool                  `json:"hoverProvider"`
	ReferencesProvider     bool                  `json:"referencesProvider"`
	RenameOptions          RenameOptions         `json:"renameOptions"`
	SignatureHelpProvider  SignatureHelpOptions  `json:"signatureHelpProvider"`
	TextDocumentSync       int                   `json:"textDocumentSync"`
	TypeDefinitionProvider bool                  `json:"typeDefinitionProvider"`
	Workspace              WorkspaceCapabilities `json:"workspace"`
}

type WorkspaceCapabilities struct {
	WorkspaceFolders WorkspaceFolderCapabilities `json:"workspaceFolders"`
}

type WorkspaceFolderCapabilities struct {
	Supported bool `json:"supported"`
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
