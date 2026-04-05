package commands

import (
	"errors"
	"io"
	"net/url"
	"path"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ViewTcb(writer io.Writer, state *parser.State, args *any) (map[string]utils.TextEdits, error) {
	changes := map[string]utils.TextEdits{}
	slice, ok := (*args).([]any)
	if !ok {
		return changes, errors.New("The args aren't an array")
	}
	if len(slice) != 1 {
		return changes, errors.New("the slice does not contain exactly one element")
	}

	uri, ok := slice[0].(string)
	if !ok {
		return changes, errors.New("The URI is not a string")
	}

	parsedUrl, err := url.Parse(uri)
	if err != nil {
		return changes, err
	}
	parsedUrl.Path = strings.TrimSuffix(parsedUrl.Path, path.Ext(parsedUrl.Path)) + interfaces.TCB_FILENAME_SUFFIX

	tcbUrl := parsedUrl.String()

	notification := interfaces.ShowDocumentNotification{
		Notification: interfaces.Notification{
			RPC:    "2.0",
			Method: "window/showDocument",
		},
		Params: interfaces.ShowDocumentParams{Uri: tcbUrl},
	}

	utils.WriteResponse(writer, notification)

	return changes, nil
}
