package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"ts_inspector/analysis/cfg"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func SaveDotForCfg(writer *utils.Writer, state *parser.State, args *any) (map[string]utils.TextEdits, error) {
	changes := map[string]utils.TextEdits{}
	uri, ok := (*args).(string)

	if !ok {
		return changes, errors.New("URI is not a string")
	}

	file, found := state.GetFile(parser.FilenameFromUri(uri))
	if !found {
		return map[string]utils.TextEdits{}, nil
	}

	cfgState, err := cfg.BuildGraphFromFile(file)
	if err != nil {
		return changes, err
	}

	sb := strings.Builder{}
	visited := map[*cfg.Block]any{}
	cfgState.PrintFromState(&sb, &visited)

	savePath := filepath.Base(file.Filename()) + "_cfg.dot"
	os.WriteFile(savePath, []byte(sb.String()), 0644)

	notification := interfaces.BuildMessageNotification("Saved "+savePath, interfaces.MessageType.Info)
	utils.WriteResponse(writer, notification)

	return map[string]utils.TextEdits{}, nil
}
