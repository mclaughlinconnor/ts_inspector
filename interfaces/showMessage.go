package interfaces

type messageType struct {
	Error int

	Warning int

	Info int

	Log int

	Debug int
}

var MessageType = messageType{1, 2, 3, 4, 5}

type ShowMessageParams struct {
	Message string `json:"message"`

	Type int `json:"type"`
}

type ShowMessageNotification struct {
	Notification
	Params ShowMessageParams `json:"params"`
}
