package interfaces

type Request[T any] struct {
	RPC    string `json:"jsonrpc"`
	ID     T      `json:"id"`
	Method string `json:"method"`
}

type Response struct {
	RPC string `json:"jsonrpc"`
	ID  *int   `json:"id,omitempty"`
}

type Notification struct {
	RPC    string `json:"jsonrpc"`
	Method string `json:"method"`
}

type EmptyResponse struct {
	Response
	Result *string `json:"result"`
}
