package actions

import (
	"os"
	"path/filepath"
	"strings"
	"ts_inspector/analysis/cfg"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func SaveDotForCfg(
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits utils.TextEdits, allowed bool, err error) {
	cfgState := cfg.BuildGraph(file)

	sb := strings.Builder{}
	visited := map[*cfg.Block]any{}
	cfgState.PrintFromState(&sb, &visited)

	os.WriteFile(filepath.Base(file.Filename())+"_cfg.dot", []byte(sb.String()), 0644)

	r := utils.Range{Start: utils.Position{Line: 0, Character: 0}, End: utils.Position{Line: 0, Character: 0}}

	return utils.TextEdits{{Range: r, NewText: ""}}, true, nil
}
