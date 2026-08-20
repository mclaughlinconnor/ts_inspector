package interfaces

type workDoneProgressKind struct {
	Begin  string
	Report string
	End    string
}

var WorkDoneProgressKind = workDoneProgressKind{"begin", "report", "end"}

type WorkDoneProgress struct {
	Kind string `json:"kind"`

	// Mandatory title of the progress operation. Used to briefly inform about
	// the kind of operation being performed.
	//
	// Examples: "Indexing" or "Linking dependencies".
	//
	Title string `json:"title"`

	// Controls if a cancel button should be shown to allow the user to cancel
	// the long running operation. Clients that don't support cancellation are
	// allowed to ignore the setting.
	Cancellable *bool `json:"cancellable,omitempty"`

	// Optional, more detailed associated progress message. Contains
	// complementary information to the `title`.
	//
	// Examples: "3/25 files", "project/src/module2", "node_modules/some_dep".
	// If unset, the previous progress message (if any) is still valid.
	Message *string `json:"message,omitempty"`

	// Optional progress percentage to display (value 100 is considered 100%).
	// If not provided infinite progress is assumed and clients are allowed
	// to ignore the `percentage` value in subsequent report notifications.
	//
	// The value should be steadily rising. Clients are free to ignore values
	// that are not following this rule. The value range is [0, 100].
	Percentage *int `json:"percentage,omitempty"`
}

type WorkDoneProgressNotification struct {
	Notification
	Params WorkDoneProgress `json:"params"`
}

type ProgressToken = int

type WorkDoneProgressCreateParams struct {
	// The token to be used to report progress.
	Token ProgressToken `json:"token"`
}

type WorkDoneProgressCreateRequest struct {
	RequestMessage
	Params WorkDoneProgressCreateParams `json:"params"`
}

type WorkDoneProgressCancelParams struct {
	// The token to be used to report progress.
	Token ProgressToken `json:"token"`
}

type WorkDoneProgressCancelRequest struct {
	RequestMessage
	Params WorkDoneProgressCancelParams `json:"params"`
}

type ProgressParams struct {
	Token ProgressToken    `json:"token"`
	Value WorkDoneProgress `json:"value"`
}

type ProgressNotification struct {
	Notification
	Params ProgressParams `json:"params"`
}
