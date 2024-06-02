package interfaces

import "ts_inspector/utils"

type completionTriggerKind struct {
	Invoked                         int
	TriggerCharacter                int
	TriggerForIncompleteCompletions int
}

var CompletionTriggerKind = completionTriggerKind{0, 1, 2}

type CompletionContext struct {
	TriggerKind      completionTriggerKind `json:"triggerKind"`
	TriggerCharacter *string               `json:"triggerCharacter,omitempty"`
}

type CompletionParams struct {
	Context *CompletionContext `json:"context,omitempty"`
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
	Data                *any                        `json:"data,omitempty"`
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

type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments *any   `json:"arguments,omitempty"`
}

type CompletionResponse struct {
	Response
	Result []CompletionItem `json:"result"` // no CompletionItemList here
}
