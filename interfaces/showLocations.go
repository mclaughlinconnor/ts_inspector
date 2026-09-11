package interfaces

type ShowLocationsNotification struct {
	Notification
	Params []Location `json:"params"`
}
