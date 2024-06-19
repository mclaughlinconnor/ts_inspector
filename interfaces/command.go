package interfaces

type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments *any   `json:"arguments,omitempty"`
}

type ExecuteCommandRequest struct {
	Request
	Params ExecuteCommandParams `json:"params"`
}

type ExecuteCommandParams struct {
	Command   string `json:"command"`
	Arguments *[]any `json:"arguments"`
}

type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}
