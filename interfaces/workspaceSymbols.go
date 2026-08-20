package interfaces

type WorkspaceSymbolRequest struct {
	RequestMessage
	Params WorkspaceSymbolParams `json:"params"`
}

type WorkspaceSymbolParams struct {
	Query string `json:"query,omitempty"`
}

type symbolTag struct {
	Deprecated int
}

var SymbolTag = symbolTag{1}

type TSymbolTag = int

type symbolKind struct {
	File          int
	Module        int
	Namespace     int
	Package       int
	Class         int
	Method        int
	Property      int
	Field         int
	Constructor   int
	Enum          int
	Interface     int
	Function      int
	Variable      int
	Constant      int
	String        int
	Number        int
	Boolean       int
	Array         int
	Object        int
	Key           int
	Null          int
	EnumMember    int
	Struct        int
	Event         int
	Operator      int
	TypeParameter int
}

var SymbolKind = symbolKind{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26}

type TSymbolKind = int

type WorkspaceSymbol struct {
	Name          string        `json:"name"`
	Kind          TSymbolKind   `json:"kind"`
	Tags          *[]TSymbolTag `json:"tags,omitempty"`
	ContainerName *string       `json:"containerName,omitempty"`
	Location      Location      `json:"location"`
	Data          *any          `json:"data,omitempty"`
}

type WorkspaceSymbolResponse struct {
	ResponseMessage
	Result []WorkspaceSymbol `json:"result"`
}
