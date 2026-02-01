package interfaces

type messageType struct {
	Error TMessageType

	Warning TMessageType

	Info TMessageType

	Log TMessageType

	Debug TMessageType
}

type TMessageType int

var MessageType = messageType{1, 2, 3, 4, 5}

type ShowMessageParams struct {
	Message string `json:"message"`

	Type TMessageType `json:"type"`
}

type ShowMessageNotification struct {
	Notification
	Params ShowMessageParams `json:"params"`
}

func BuildMessageNotification(message string, messageType TMessageType) *ShowMessageNotification {
	return &ShowMessageNotification{
		Notification: Notification{RPC: "2.0", Method: "window/showMessage"},
		Params:       ShowMessageParams{Type: messageType, Message: message},
	}
}
