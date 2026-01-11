package actions

import (
	"io"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func CalculateAllProviders(
	writer io.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits utils.TextEdits, allowed bool, err error) {
	allProviders := []string{}

	for _, class := range file.Snapshot().Classes {
		for _, p := range class.GetAllProvidedValues(state) {
			content := []byte(p.Source.Snapshot().File.Snapshot().Content)

			text := p.Provider.Token.Name

			if p.Provider.Class != nil {
				text = text + ": " + p.Provider.Class.Name
			} else if p.Provider.Existing != nil {
				text = text + ": " + p.Provider.Existing.Name
			} else if p.Provider.Factory != nil {
				text = text + ": " + p.Provider.Factory.Content(content)
			} else if p.Provider.RefToken != nil {
				text = text + ": " + p.Provider.RefToken.Name
			} else if p.Provider.Value != nil {
				text = text + ": " + p.Provider.Value.Content(content)
			}

			allProviders = append(allProviders, text)
		}
	}

	notification := interfaces.ShowMessageNotification{
		Notification: interfaces.Notification{
			RPC:    "2.0",
			Method: "window/showMessage",
		},
		Params: interfaces.ShowMessageParams{Type: interfaces.MessageType.Info, Message: strings.Join(allProviders, ", ")},
	}

	utils.WriteResponse(writer, notification)

	r := utils.Range{Start: utils.Position{Line: 0, Character: 0}, End: utils.Position{Line: 0, Character: 0}}

	return utils.TextEdits{{Range: r, NewText: ""}}, true, nil
}
