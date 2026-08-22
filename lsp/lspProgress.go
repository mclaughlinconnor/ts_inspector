package lsp

import (
	"fmt"
	"log"
	"ts_inspector/interfaces"
	"ts_inspector/utils"
)

var progressResponses map[int]chan bool = map[int]chan bool{}

func lspCreateProgressToken(writer *utils.Writer) (*interfaces.ProgressToken, bool) {
	id := utils.GetNextId()
	token := utils.GetNextId()

	request := interfaces.WorkDoneProgressCreateRequest{
		RequestMessage: interfaces.RequestMessage{
			RPC:    "2.0",
			ID:     id,
			Method: "window/workDoneProgress/create",
		},
		Params: interfaces.WorkDoneProgressCreateParams{
			Token: token,
		},
	}

	lspIdHandler[id] = func(writer *utils.Writer, logger *log.Logger, contents []byte) {
		lspHandleTokenCreateResponse(writer, logger, contents)
		lspIdHandler[id] = nil
	}

	c := make(chan bool, 1)
	progressResponses[id] = c

	utils.WriteResponse(writer, request)

	select {
	case <-Shutdown:
		return nil, false
	case result := <-c:
		if !result {
			return nil, false
		}

		return &id, true
	}
}

func lspHandleTokenCreateResponse(writer *utils.Writer, logger *log.Logger, contents []byte) {
	response := utils.TryParseRequest[interfaces.ResponseMessage](logger, contents)

	id := response.ID
	if response.ID == nil {
		utils.WriteResponse(writer, interfaces.BuildMessageNotification("Got token create response without ID", interfaces.MessageType.Warning))
		return
	}

	c, found := progressResponses[*id]
	if !found {
		utils.WriteResponse(writer, interfaces.BuildMessageNotification(fmt.Sprintf("Got token create response for %v, but I don't have that request ID", *id), interfaces.MessageType.Warning))
	}

	if response.Error != nil {
		utils.WriteResponse(writer, interfaces.BuildMessageNotification(response.Error.Message, interfaces.MessageType.Error))
		c <- false
		return
	}

	c <- true
}

func lspBeginProgress(writer *utils.Writer, progressToken *interfaces.ProgressToken, title string, message string, percentage int) {
	if progressToken == nil {
		return
	}

	var m *string = nil
	if message != "" {
		m = &message
	}

	var p *int = nil
	if percentage != -1 {
		p = &percentage
	}

	progress := interfaces.WorkDoneProgress{
		Kind:       interfaces.WorkDoneProgressKind.Begin,
		Title:      title,
		Message:    m,
		Percentage: p,
	}

	sendProgressNotification(writer, *progressToken, progress)
}

func lspReportProgress(writer *utils.Writer, progressToken *interfaces.ProgressToken, message string, percentage int) {
	if progressToken == nil {
		return
	}

	var m *string = nil
	if message != "" {
		m = &message
	}

	var p *int = nil
	if percentage != -1 {
		p = &percentage
	}

	progress := interfaces.WorkDoneProgress{
		Kind:       interfaces.WorkDoneProgressKind.Report,
		Message:    m,
		Percentage: p,
	}

	sendProgressNotification(writer, *progressToken, progress)
}

func lspEndProgress(writer *utils.Writer, progressToken *interfaces.ProgressToken, message string) {
	if progressToken == nil {
		return
	}

	var m *string = nil
	if message != "" {
		m = &message
	}

	progress := interfaces.WorkDoneProgress{
		Kind:    interfaces.WorkDoneProgressKind.End,
		Message: m,
	}

	sendProgressNotification(writer, *progressToken, progress)
}

func sendProgressNotification(writer *utils.Writer, progressToken interfaces.ProgressToken, progress interfaces.WorkDoneProgress) {
	notification := interfaces.ProgressNotification{
		Notification: interfaces.Notification{RPC: "2.0", Method: "$/progress"},
		Params:       interfaces.ProgressParams{Token: progressToken, Value: progress},
	}

	utils.WriteResponse(writer, notification)
}
