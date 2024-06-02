package ngserver

func SendToAngular(message string) {
	inputMessages <- message
}

func ReceiveFromAngular() string {
	m := <-outputMessages
	return m
}
