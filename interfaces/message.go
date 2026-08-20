package interfaces

type RequestMessage struct {
	RPC    string `json:"jsonrpc"`
	ID     int    `json:"id"`
	Method string `json:"method"`
}

type ResponseMessage struct {
	RPC   string         `json:"jsonrpc"`
	ID    *int           `json:"id,omitempty"`
	Error *ResponseError `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

const (
	// Defined by JSON-RPC
	ErrorCodesParseError     = -32700
	ErrorCodesInvalidRequest = -32600
	ErrorCodesMethodNotFound = -32601
	ErrorCodesInvalidParams  = -32602
	ErrorCodesInternalError  = -32603

	// This is the start range of JSON-RPC reserved error codes.
	// It doesn't denote a real error code. No LSP error codes should
	// be defined between the start and end range. For backwards
	// compatibility the `ServerNotInitialized` and the `UnknownErrorCode`
	// are left in the range.
	//
	// @since 3.16.0
	ErrorCodesJsonrpcReservedErrorRangeStart = -32099
	// @deprecated use jsonrpcReservedErrorRangeStart
	ErrorCodesServerErrorStart = ErrorCodesJsonrpcReservedErrorRangeStart

	// Error code indicating that a server received a notification or
	// request before the server received the `initialize` request.
	ErrorCodesServerNotInitialized = -32002
	ErrorCodesUnknownErrorCode     = -32001

	// This is the end range of JSON-RPC reserved error codes.
	// It doesn't denote a real error code.
	//
	// @since 3.16.0
	ErrorCodesJsonrpcReservedErrorRangeEnd = -32000
	// @deprecated use jsonrpcReservedErrorRangeEnd
	ErrorCodesServerErrorEnd = ErrorCodesJsonrpcReservedErrorRangeEnd

	// This is the start range of LSP reserved error codes.
	// It doesn't denote a real error code.
	//
	// @since 3.16.0
	ErrorCodesLspReservedErrorRangeStart = -32899

	// A request failed but it was syntactically correct, e.g the
	// method name was known and the parameters were valid. The error
	// message should contain human readable information about why
	// the request failed.
	//
	// @since 3.17.0
	ErrorCodesRequestFailed = -32803

	// The server cancelled the request. This error code should
	// only be used for requests that explicitly support being
	// server cancellable.
	//
	// @since 3.17.0
	ErrorCodesServerCancelled = -32802

	// The server detected that the content of a document got
	// modified outside normal conditions. A server should
	// NOT send this error code if it detects a content change
	// in its unprocessed messages. The result even computed
	// on an older state might still be useful for the client.
	//
	// If a client decides that a result is not of any use anymore
	// the client should cancel the request.
	ErrorCodesContentModified = -32801

	// The client has canceled a request and a server has detected
	// the cancel.
	ErrorCodesRequestCancelled = -32800

	// This is the end range of LSP reserved error codes.
	// It doesn't denote a real error code.
	//
	// @since 3.16.0
	ErrorCodesLspReservedErrorRangeEnd = -32800
)

type Notification struct {
	RPC    string `son:"jsonrpc"`
	Method string `json:"method"`
}

type EmptyResponse struct {
	ResponseMessage
	Result *string `json:"result"`
}
