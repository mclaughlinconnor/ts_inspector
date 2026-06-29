package parser

import (
	"fmt"
	"strconv"
	"strings"
)

type (
	SnapshotID  uint64
	ProjectID   string
	SymbolID    uint64
	TypeID      uint32
	SignatureID uint64
	NodeHandle  string
)

type Node struct {
	Pos  int
	End  int
	Kind int // no idea what this is yet
	Path string
}

func (h NodeHandle) ExtractNode() (*Node, error) {
	e := func() (*Node, error) {
		return nil, fmt.Errorf("invalid node handle %q", h)
	}

	parts := strings.SplitN(string(h), ".", 4)
	if len(parts) != 4 {
		return e()
	}

	n := Node{}

	pos, err := strconv.Atoi(parts[0])
	if err != nil {
		return e()
	}
	n.Pos = pos

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return e()
	}
	n.End = end

	kind, err := strconv.Atoi(parts[2])
	if err != nil {
		return e()
	}
	n.Kind = kind

	n.Path = parts[3]

	return &n, nil
}

type TsGoResponse struct {
	RPC string `json:"jsonrpc"`
	ID  string `json:"id"`
}

type TsGoRequest struct {
	RPC    string `json:"jsonrpc"`
	ID     string `json:"id"`
	Method string `json:"method"`
}

// InitializeResponse is returned by the initialize method.
type InitializeResponse struct {
	TsGoResponse
	Result struct {
		// UseCaseSensitiveFileNames indicates whether the host file system is case-sensitive.
		UseCaseSensitiveFileNames bool `json:"useCaseSensitiveFileNames"`
		// CurrentDirectory is the server's current working directory.
		CurrentDirectory string `json:"currentDirectory"`
	}
}

// UpdateSnapshotParams are the parameters for creating a new snapshot.
// All fields are optional. With no fields set, the server adopts the latest LSP state.
type UpdateSnapshotParams struct {
	// OpenProject is the path to a tsconfig.json file to open/load in the new snapshot.
	OpenProjects *[]string `json:"openProjects,omitempty"`
	// FileChanges describes file system changes since the last snapshot.
	FileChanges *APIFileChanges `json:"fileChanges,omitempty"`
}

type UpdateSnapshotRequest struct {
	TsGoRequest
	Params UpdateSnapshotParams `json:"params"`
}

type CompilerOptions = any

type ProjectResponse struct {
	Id              ProjectID        `json:"id"`
	ConfigFileName  string           `json:"configFileName"`
	RootFiles       []string         `json:"rootFiles"`
	CompilerOptions *CompilerOptions `json:"compilerOptions"`
}

// UpdateSnapshotResponse is returned by updateSnapshot.
type UpdateSnapshotResponse struct {
	TsGoResponse
	Result struct {
		// Snapshot is the handle for the newly created snapshot.
		Snapshot SnapshotID `json:"snapshot"`
		// Projects is the list of projects in the snapshot.
		Projects []*ProjectResponse `json:"projects"`
		// Changes describes source file differences from the previous snapshot.
		// Nil for the first snapshot in a session.
		Changes *SnapshotChanges `json:"changes,omitempty"`
	}
}

type Path string

// ProjectFileChanges describes what source files changed within a single project.
type ProjectFileChanges struct {
	// ChangedFiles lists source file paths whose content differs.
	ChangedFiles []Path `json:"changedFiles,omitempty"`
	// DeletedFiles lists source file paths removed from the project's program.
	DeletedFiles []Path `json:"deletedFiles,omitempty"`
}

// SnapshotChanges describes what changed between the previous latest snapshot
// and the newly created snapshot. Changes are reported per-project so clients
// can track cache refs at the (snapshot, project) level.
type SnapshotChanges struct {
	// ChangedProjects maps project handles to the file changes within that project.
	// Projects not listed here (and not in RemovedProjects) are unchanged.
	ChangedProjects map[ProjectID]*ProjectFileChanges `json:"changedProjects,omitempty"`
	// RemovedProjects lists project handles that were present in the previous
	// snapshot but absent from the new one.
	RemovedProjects []ProjectID `json:"removedProjects,omitempty"`
}

// APIFileChanges describes file changes to apply when updating a snapshot.
// Either InvalidateAll is true (discard all caches) or Changed/Created/Deleted
// list individual documents.
type APIFileChanges struct {
	InvalidateAll bool                 `json:"invalidateAll,omitempty"`
	Changed       []DocumentIdentifier `json:"changed,omitempty"`
	Created       []DocumentIdentifier `json:"created,omitempty"`
	Deleted       []DocumentIdentifier `json:"deleted,omitempty"`
}

// DocumentIdentifier identifies a document by either a file name (plain string) or a URI object.
// On the wire it is string | { uri: string }.
type DocumentIdentifier struct {
	FileName string `json:"fileName,omitempty"`
	URI      string `json:"uri,omitempty"`
}

type ReadFileRequest struct {
	TsGoRequest
	Params string `json:"params"`
}

type FileExistsRequest struct {
	TsGoRequest
	Params string `json:"params"`
}

type GetAccessibleEntriesRequest struct {
	TsGoRequest
	Params string `json:"params"`
}

type Content struct {
	Content string `json:"content"`
}

type ReadFileResponse struct {
	TsGoResponse
	Result *Content `json:"result"`
}

type FileExistsResponse struct {
	TsGoResponse
	Result *bool `json:"result"`
}

type Entries struct {
	Files       []string `json:"files"`
	Directories []string `json:"directories"`
}

type GetAcceessibleEntriesResponse struct {
	TsGoResponse
	Result *Entries `json:"result"`
}

type GetDiagnosticsParams struct {
	Snapshot SnapshotID          `json:"snapshot"`
	Project  ProjectID           `json:"project"`
	File     *DocumentIdentifier `json:"file,omitempty"`
}

type GetDiagnosticsRequest struct {
	TsGoRequest
	Params GetDiagnosticsParams `json:"params"`
}

type Diagnostic struct {
	// FileName is the path of the file this diagnostic belongs to, if any.
	FileName string `json:"fileName,omitempty"`
	// Pos is the start position of the diagnostic in the source file.
	Pos int `json:"pos"`
	// End is the end position of the diagnostic in the source file.
	End int `json:"end"`
	// Code is the diagnostic error code.
	Code int32 `json:"code"`
	// Category is the diagnostic category (error, warning, suggestion, message).
	Category Category `json:"category"`
	// Text is the localized diagnostic message text.
	Text string `json:"text"`
	// ReportsUnnecessary indicates this diagnostic highlights unnecessary code.
	ReportsUnnecessary bool `json:"reportsUnnecessary,omitzero"`
	// ReportsDeprecated indicates this diagnostic highlights deprecated code.
	ReportsDeprecated bool `json:"reportsDeprecated,omitzero"`
	// MessageChain contains chained diagnostic messages, if any.
	MessageChain []*Diagnostic `json:"messageChain,omitempty"`
	// RelatedInformation contains related diagnostic information, if any.
	RelatedInformation []*Diagnostic `json:"relatedInformation,omitempty"`
}

// DiagnosticResponse is the API response for a single diagnostic.
type DiagnosticResponse struct {
	Result []Diagnostic
}

type Category int32

const (
	CategoryWarning Category = iota
	CategoryError
	CategorySuggestion
	CategoryMessage
)

func (category Category) Name() string {
	switch category {
	case CategoryWarning:
		return "warning"
	case CategoryError:
		return "error"
	case CategorySuggestion:
		return "suggestion"
	case CategoryMessage:
		return "message"
	}
	panic("Unhandled diagnostic category")
}

type GetSymbolAtPositionParams struct {
	Snapshot SnapshotID         `json:"snapshot"`
	Project  ProjectID          `json:"project"`
	File     DocumentIdentifier `json:"file"`
	Position uint32             `json:"position"`
}

type GetSymbolAtPositionRequest struct {
	TsGoRequest
	Params GetSymbolAtPositionParams `json:"params"`
}

type SymbolResponse struct {
	TsGoResponse
	Result struct {
		Id               SymbolID     `json:"id"`
		Name             string       `json:"name"`
		Flags            uint32       `json:"flags"`
		CheckFlags       uint32       `json:"checkFlags"`
		Declarations     []NodeHandle `json:"declarations,omitempty"`
		ValueDeclaration NodeHandle   `json:"valueDeclaration,omitempty"`
	}
}

type GetTypeOfSymbolRequest struct {
	TsGoRequest
	Params GetTypeOfSymbolParams `json:"params"`
}

type GetTypeOfSymbolParams struct {
	Snapshot SnapshotID `json:"snapshot"`
	Project  ProjectID  `json:"project"`
	Symbol   SymbolID   `json:"symbol"`
}

type GetTypeAtPositionParams struct {
	Snapshot SnapshotID         `json:"snapshot"`
	Project  ProjectID          `json:"project"`
	File     DocumentIdentifier `json:"file"`
	Position uint32             `json:"position"`
}

type GetTypeAtPositionParamsRequest struct {
	TsGoRequest
	Params GetTypeAtPositionParams `json:"params"`
}

type TypeResponse struct {
	TsGoResponse
	Result struct {
		Id          TypeID `json:"id"`
		Flags       uint32 `json:"flags"`
		ObjectFlags uint32 `json:"objectFlags,omitempty"`

		// LiteralType data
		Value any `json:"value,omitempty"`

		// ObjectType / TypeReference / StringMappingType / IndexType target
		Target NodeHandle `json:"target,omitempty"`

		// InterfaceType type parameters
		TypeParameters      []TypeID `json:"typeParameters,omitempty"`
		OuterTypeParameters []TypeID `json:"outerTypeParameters,omitempty"`
		LocalTypeParameters []TypeID `json:"localTypeParameters,omitempty"`

		// TupleType data
		ElementFlags  []ElementFlags `json:"elementFlags,omitempty"`
		FixedLength   *int           `json:"fixedLength,omitempty"`
		TupleReadonly *bool          `json:"readonly,omitempty"`

		// IndexedAccessType data
		ObjectType TypeID `json:"objectType,omitempty"`
		IndexType  TypeID `json:"indexType,omitempty"`

		// ConditionalType data
		CheckType   TypeID `json:"checkType,omitempty"`
		ExtendsType TypeID `json:"extendsType,omitempty"`

		// SubstitutionType data
		BaseType        TypeID `json:"baseType,omitempty"`
		SubstConstraint TypeID `json:"substConstraint,omitempty"`

		// TemplateLiteralType text segments
		Texts []string `json:"texts,omitempty"`

		// Symbol associated with structured types
		Symbol SymbolID `json:"symbol,omitempty"`
	}
}

type ElementFlags uint32

const (
	ElementFlagsNone        ElementFlags = 0
	ElementFlagsRequired    ElementFlags = 1 << 0 // T
	ElementFlagsOptional    ElementFlags = 1 << 1 // T?
	ElementFlagsRest        ElementFlags = 1 << 2 // ...T[]
	ElementFlagsVariadic    ElementFlags = 1 << 3 // ...T
	ElementFlagsFixed                    = ElementFlagsRequired | ElementFlagsOptional
	ElementFlagsVariable                 = ElementFlagsRest | ElementFlagsVariadic
	ElementFlagsNonRequired              = ElementFlagsOptional | ElementFlagsRest | ElementFlagsVariadic
	ElementFlagsNonRest                  = ElementFlagsRequired | ElementFlagsOptional | ElementFlagsVariadic
)

type TypeFormatFlags uint32

const (
	TypeFormatFlagsNone                               TypeFormatFlags = 0
	TypeFormatFlagsNoTruncation                       TypeFormatFlags = 1 << 0 // Don't truncate typeToString result
	TypeFormatFlagsWriteArrayAsGenericType            TypeFormatFlags = 1 << 1 // Write Array<T> instead T[]
	TypeFormatFlagsGenerateNamesForShadowedTypeParams TypeFormatFlags = 1 << 2 // When a type parameter T is shadowing another T, generate a name for it so it can still be referenced
	TypeFormatFlagsUseStructuralFallback              TypeFormatFlags = 1 << 3 // When an alias cannot be named by its symbol, rather than report an error, fallback to a structural printout if possible
	// hole because there's a hole in node builder flags
	TypeFormatFlagsWriteTypeArgumentsOfSignature TypeFormatFlags = 1 << 5 // Write the type arguments instead of type parameters of the signature
	TypeFormatFlagsUseFullyQualifiedType         TypeFormatFlags = 1 << 6 // Write out the fully qualified type name (eg. Module.Type, instead of Type)
	// hole because `UseOnlyExternalAliasing` is here in node builder flags, but functions which take old flags use `SymbolFormatFlags` instead
	TypeFormatFlagsSuppressAnyReturnType TypeFormatFlags = 1 << 8 // If the return type is any-like, don't offer a return type.
	// hole because `WriteTypeParametersInQualifiedName` is here in node builder flags, but functions which take old flags use `SymbolFormatFlags` for this instead
	TypeFormatFlagsMultilineObjectLiterals             TypeFormatFlags = 1 << 10 // Always print object literals across multiple lines (only used to map into node builder flags)
	TypeFormatFlagsWriteClassExpressionAsTypeLiteral   TypeFormatFlags = 1 << 11 // Write a type literal instead of (Anonymous class)
	TypeFormatFlagsUseTypeOfFunction                   TypeFormatFlags = 1 << 12 // Write typeof instead of function type literal
	TypeFormatFlagsOmitParameterModifiers              TypeFormatFlags = 1 << 13 // Omit modifiers on parameters
	TypeFormatFlagsUseAliasDefinedOutsideCurrentScope  TypeFormatFlags = 1 << 14 // For a `type T = ... ` defined in a different file, write `T` instead of its value, even though `T` can't be accessed in the current scope.
	TypeFormatFlagsUseSingleQuotesForStringLiteralType TypeFormatFlags = 1 << 28 // Use single quotes for string literal type
	TypeFormatFlagsNoTypeReduction                     TypeFormatFlags = 1 << 29 // Don't call getReducedType
	TypeFormatFlagsUseInstantiationExpressions         TypeFormatFlags = 1 << 30 // Use instantiation expressions for qualified instantiated names like Foo<string>.Bar
	TypeFormatFlagsOmitThisParameter                   TypeFormatFlags = 1 << 25
	TypeFormatFlagsWriteCallStyleSignature             TypeFormatFlags = 1 << 27 // Write construct signatures as call style signatures
	// Error Handling
	TypeFormatFlagsAllowUniqueESSymbolType TypeFormatFlags = 1 << 20 // This is bit 20 to align with the same bit in `NodeBuilderFlags`
	// TypeFormatFlags exclusive
	TypeFormatFlagsAddUndefined             TypeFormatFlags = 1 << 17 // Add undefined to types of initialized, non-optional parameters
	TypeFormatFlagsWriteArrowStyleSignature TypeFormatFlags = 1 << 18 // Write arrow style signature
	// State
	TypeFormatFlagsInArrayType         TypeFormatFlags = 1 << 19 // Writing an array element type
	TypeFormatFlagsInElementType       TypeFormatFlags = 1 << 21 // Writing an array or union element type
	TypeFormatFlagsInFirstTypeArgument TypeFormatFlags = 1 << 22 // Writing first type argument of the instantiated type
	TypeFormatFlagsInTypeAlias         TypeFormatFlags = 1 << 23 // Writing type in type alias declaration
)

// TypeToTypeNodeParams are the parameters for the typeToTypeNode method.
type TypeToTypeNodeParams struct {
	Snapshot SnapshotID      `json:"snapshot"`
	Project  ProjectID       `json:"project"`
	Type     TypeID          `json:"type"`
	Location NodeHandle      `json:"location,omitempty"`
	Flags    TypeFormatFlags `json:"flags,omitempty"`
}

type TypeToTypeNodeRequest struct {
	TsGoRequest
	Params TypeToTypeNodeParams `json:"params"`
}

type TypeToStringResponse struct {
	TsGoResponse
	Result string `json:"result"`
}
