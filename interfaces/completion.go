package interfaces

import "ts_inspector/utils"

type completionTriggerKind struct {
	Invoked                         int
	TriggerCharacter                int
	TriggerForIncompleteCompletions int
}

var CompletionTriggerKind = completionTriggerKind{0, 1, 2}

type CompletionContext struct {
	TriggerKind      int     `json:"triggerKind"`
	TriggerCharacter *string `json:"triggerCharacter,omitempty"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     utils.Position         `json:"position"`
	Context      *CompletionContext     `json:"context,omitempty"`
}

type CompletionRequest struct {
	Request
	Params CompletionParams `json:"params"`
}

type CompletionOptions struct {
	TriggerCharacters   *[]string `json:"triggerCharacters,omitempty"`
	AllCommitCharacters *[]string `json:"allCommitCharacters,omitempty"`
	ResolveProvider     *bool     `json:"resolveProvider,omitempty"`
	CompletionItem      *struct {
		LabelDetailsSupport *bool `json:"labelDetailsSupport,omitempty"`
	} `json:"completionItem,omitempty"`
}

type CompletionItemRequest struct {
	Request
	Params CompletionItem `json:"params"`
}

type CompletionItem struct {
	Label               string                      `json:"label"`
	LabelDetails        *CompletionItemLabelDetails `json:"labelDetails,omitempty"`
	Kind                *int                        `json:"kind,omitempty"`
	Tags                *[]int                      `json:"tags,omitempty"`
	Detail              *string                     `json:"detail,omitempty"`
	Documentation       *string                     `json:"documentation,omitempty"`
	Deprecated          *bool                       `json:"deprecated,omitempty"`
	Preselect           *bool                       `json:"preselect,omitempty"`
	SortText            *string                     `json:"sortText,omitempty"`
	FilterText          *string                     `json:"filterText,omitempty"`
	InsertText          *string                     `json:"insertText,omitempty"`
	InsertTextFormat    *int                        `json:"insertTextFormat,omitempty"`
	InsertTextMode      *int                        `json:"insertTextMode,omitempty"`
	TextEdit            *TextEdit                   `json:"textEdit,omitempty"`
	TextEditText        *string                     `json:"textEditText,omitempty"`
	AdditionalTextEdits *[]TextEdit                 `json:"additionalTextEdits,omitempty"`
	CommitCharacters    *[]string                   `json:"commitCharacters,omitempty"`
	Command             *Command                    `json:"command,omitempty"`
	Data                map[string]any              `json:"data,omitempty"`
}

type CompletionItemLabelDetails struct {
	Detail      string `json:"detail"`
	Description string `json:"description"`
}

type InsertReplaceEdit struct {
	NewText string      `json:"newText"`
	Insert  utils.Range `json:"insert"`
	Replace utils.Range `json:"replace"`
}

type completionItemKind struct {
	Text          int
	Method        int
	Function      int
	Constructor   int
	Field         int
	Variable      int
	Class         int
	Interface     int
	Module        int
	Property      int
	Unit          int
	Value         int
	Enum          int
	Keyword       int
	Snippet       int
	Color         int
	File          int
	Reference     int
	Folder        int
	EnumMember    int
	Constant      int
	Struct        int
	Event         int
	Operator      int
	TypeParameter int
}

var CompletionItemKind = completionItemKind{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}

type completionItemTag struct {
	Deprecated int
}

var CompletionItemTag = completionItemTag{1}

type insertTextFormat struct {
	PlainText int
	Snippet   int
}

var InsertTextFormat = insertTextFormat{1, 2}

type insertTextMode struct {
	AsIs              int `json:"asIs"`
	AdjustIndentation int `json:"adjustIndentation"`
}

var InsertTextMode = insertTextMode{1, 2}

type TextEdit struct {
	Range   utils.Range `json:"Range"`
	NewText string      `json:"NewText"`
}

type CompletionResponse struct {
	Response
	Result []CompletionItem `json:"result"` // no CompletionItemList here
}

type InlineCompletionRequest struct {
	Request
	Params InlineCompletionParams `json:"params"`
}

type inlineCompletionTriggerKind struct {
	Invoked   int
	Automatic int
}

var InlineCompletionTriggerKind = inlineCompletionTriggerKind{1, 2}

type SelectedCompletionInfo struct {
	Range utils.Range `json:"range"`
	Text  string      `json:"text"`
}

type InlineCompletionContext struct {
	TriggerKind            int `json:"triggerKind"`
	SelectedCompletionInfo any `json:"selectedCompletionInfo"` // vscode seems to be off-spec sending an array
}

type InlineCompletionParams struct {
	TextDocument TextDocumentIdentifier  `json:"textDocument"`
	Position     utils.Position          `json:"position"`
	Context      InlineCompletionContext `json:"context"`
}

type InlineCompletionItem struct {
	InsertText string       `json:"insertText"`
	FilterText *string      `json:"filterText"`
	Range      *utils.Range `json:"range"`
	Command    *Command     `json:"command"`
}

type InlineCompletionResponse struct {
	Response
	Result []InlineCompletionItem `json:"result"`
}

type MLXServerCompletionResult struct {
	Id                string                            `json:"id"`
	SystemFingerprint string                            `json:"system_fingerprint"`
	Object            string                            `json:"object"`
	Model             string                            `json:"model"`
	Created           int                               `json:"created"`
	Choices           []MLXServerCompletionResultChoice `json:"choices"`
	Usage             MLXServerCompletionResultUsage    `json:"usage"`
}

type MLXServerCompletionResultUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type MLXServerCompletionResultChoice struct {
	Index        int                                     `json:"index"`
	FinishReason string                                  `json:"finish_reason"`
	LogProbs     MLXServerCompletionResultChoiceLogProbs `json:"logprobs"`
	Text         string                                  `json:"text"`
}

type MLXServerCompletionResultChoiceLogProbs struct {
	TokenLogProbs []float32 `json:"token_logprobs"`
	TopLogProbs   []float32 `json:"top_logprobs"`
	Tokens        []float32 `json:"tokens"`
}
