package interfaces

type ApplyWorkspaceEditRequest struct {
	Request
	Params ApplyWorkspaceEditParams `json:"params"`
}

type ApplyWorkspaceEditParams struct {
	Label string        `json:"label"`
	Edit  WorkspaceEdit `json:"edit"`
}

type ApplyWorkspaceEditResult struct {
	Applied       bool    `json:"applied"`
	FailureReason *string `json:"failureReason"`
	FailedChange  *uint32 `json:"failedChange"`
}
