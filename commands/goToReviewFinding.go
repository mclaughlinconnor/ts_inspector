package commands

import (
	"errors"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func GoToReviewFinding(writer *utils.Writer, state *parser.State, args *any) (map[string]utils.TextEdits, error) {
	changes := map[string]utils.TextEdits{}
	slice, ok := (*args).([]any)

	if !ok {
		return changes, errors.New("the args aren't an array")
	}

	if len(slice) != 2 {
		return changes, errors.New("the slice does not contain exactly two elements")
	}

	uri, ok1 := slice[0].(string)
	lineF, ok2 := slice[1].(float64)

	if !ok1 {
		return changes, errors.New("uri must be string")
	}

	if !ok2 {
		return changes, errors.New("line must be int")
	}

	line := int(lineF)

	file, found := state.GetFile(parser.FilenameFromUri(uri))
	if !found {
		return map[string]utils.TextEdits{}, nil
	}

	documentShown := false

	for _, finding := range state.GetReviewFindings() {
		if finding.Metadata.FilePath != file.Filename() {
			continue
		}

		if finding.Metadata.NewLine-1 != line && finding.Metadata.OldLine-1 != line {
			continue
		}

		takeFocus := true
		notification := interfaces.ShowDocumentNotification{
			Notification: interfaces.Notification{
				RPC:    "2.0",
				Method: "window/showDocument",
			},
			Params: interfaces.ShowDocumentParams{TakeFocus: &takeFocus, Uri: parser.UriFromFilename(finding.FilePath)},
		}

		utils.WriteResponse(writer, notification)

		documentShown = true

		break
	}

	if !documentShown {
		notification := interfaces.BuildMessageNotification("No review finding found", interfaces.MessageType.Info)
		utils.WriteResponse(writer, notification)
	}

	return map[string]utils.TextEdits{}, nil
}
