package analysis

import (
	"ts_inspector/analysis/cfg"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func cfgUnreachableBlock(_ *parser.State, file *parser.File) []Analysis {
	if file.Snapshot().Filetype != "typescript" {
		return []Analysis{}
	}

	analyses := []Analysis{}

	cfgState := cfg.BuildGraph(file)
	for _, cfg := range cfgState.AllCfg {
		for _, block := range cfg.Blocks {
			if len(block.Before) != 0 || cfg.Start == block {
				continue
			}

			message := "Code is unreachable"

			var node *sitter.Node
			if block.Node != nil {
				node = block.Node
			} else if len(block.Instructions) != 0 {
				node = block.Instructions[0].Node
			}

			if node == nil {
				continue
			}

			content := file.Snapshot().Content
			startPosition := utils.GetPositionForOffset(content, node.StartByte())
			endPosition := utils.GetPositionForOffset(content, node.EndByte())

			analyses = append(analyses, newAnalysis("unreachable", utils.Range{Start: startPosition, End: endPosition}, AnalysisSeverity.Error, message))
		}
	}

	return analyses
}
