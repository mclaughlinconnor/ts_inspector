package commands

import (
	"errors"
	"io"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func PrintProviders(writer io.Writer, state *parser.State, args *any) (map[string]utils.TextEdits, error) {
	changes := map[string]utils.TextEdits{}
	uri, ok := (*args).(string)

	if !ok {
		return changes, errors.New("URI is not a string")
	}

	allProviders := []string{}

	file, found := state.GetFile(parser.FilenameFromUri(uri))
	if !found {
		return map[string]utils.TextEdits{}, nil
	}

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

	return map[string]utils.TextEdits{}, nil
}
