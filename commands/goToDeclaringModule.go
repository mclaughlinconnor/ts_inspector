package commands

import (
	"errors"
	"io"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func GoToDeclaringModule(writer io.Writer, state *parser.State, args *any) (map[string]utils.TextEdits, error) {
	changes := map[string]utils.TextEdits{}
	slice, ok := (*args).([]any)

	if !ok {
		return changes, errors.New("The args aren't an array")
	}

	if len(slice) != 3 {
		return changes, errors.New("The slice does not contain exactly three elements")
	}

	uri, ok1 := slice[0].(string)
	rangeStartOffsetF, ok2 := slice[1].(float64)
	rangeEndOffsetF, ok3 := slice[2].(float64)

	if !ok1 {
		return changes, errors.New("URI must be string")
	}

	if !ok2 {
		return changes, errors.New("rangeStartOffset must be int")
	}

	if !ok3 {
		return changes, errors.New("rangeEndOffset must be int")
	}

	rangeStartOffset := uint32(rangeStartOffsetF)
	rangeEndOffset := uint32(rangeEndOffsetF)

	file, found := state.GetFile(parser.FilenameFromUri(uri))
	if !found {
		return map[string]utils.TextEdits{}, nil
	}

	documentShown := false

	for _, class := range file.Snapshot().Classes {
		node := class.Snapshot().Node
		nodeStartOffset := node.StartByte()
		nodeEndOffset := node.EndByte()

		if rangeStartOffset < nodeStartOffset || rangeStartOffset > nodeEndOffset ||
			rangeEndOffset < nodeStartOffset || rangeEndOffset > nodeEndOffset {
			continue
		}

		if !class.HasComponent() {
			continue
		}

		declaredIns := class.Snapshot().Angular.Component.DeclaredIn
		if len(declaredIns) == 0 {
			continue
		}

		declaredIn := declaredIns[0].Snapshot()

		declaringFile := declaredIn.File.Snapshot()
		declaringUri := declaringFile.URI

		classOffset := declaredIn.Node.StartByte()
		nameOffset := classOffset + declaredIn.NameNode.StartByte()

		position := parser.GetPositionForOffset(declaringFile.Content, nameOffset)
		selection := utils.Range{Start: position, End: position}

		notification := interfaces.ShowDocumentNotification{
			Notification: interfaces.Notification{
				RPC:    "2.0",
				Method: "window/showDocument",
			},
			Params: interfaces.ShowDocumentParams{Uri: declaringUri, Selection: &selection},
		}

		utils.WriteResponse(writer, notification)

		documentShown = true

		break
	}

	if !documentShown {
		notification := interfaces.ShowMessageNotification{
			Notification: interfaces.Notification{
				RPC:    "2.0",
				Method: "window/showMessage",
			},
			Params: interfaces.ShowMessageParams{Type: interfaces.MessageType.Info, Message: "No declaring module found"},
		}

		utils.WriteResponse(writer, notification)
	}

	return map[string]utils.TextEdits{}, nil
}
