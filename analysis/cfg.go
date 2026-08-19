package analysis

import (
	"ts_inspector/analysis/cfg"
	"ts_inspector/config"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

const unreachableCode = "unreachable"

func cfgUnreachableBlock(state *parser.State, file *parser.File) []Analysis {
	analyses := []Analysis{}

	if file.Snapshot().Filetype == "typescript" {
		return analyseCfg(cfg.BuildGraphFromFile(file), analyses, func(m string, n *sitter.Node, s int) *Analysis {
			a := newAnalysisFromFileNode(file, unreachableCode, n, s, m, nil)
			return &a
		})
	}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	content := []byte(file.Snapshot().Content)

	buildPugAnalysis := func(tcbBlock *tcb.Statement) func(string, *sitter.Node, int) *Analysis {
		return func(message string, node *sitter.Node, severity int) *Analysis {
			r := tcbBlock.TsNodeToRange(file.Snapshot().Content, node, config.Debug)
			if r == nil {
				return nil
			}

			a := newAnalysis(unreachableCode, *r, severity, message, nil)

			return &a
		}
	}

	for _, class := range file.Snapshot().Classes {
		root, err := utils.ParseText([]byte(content), utils.Pug)
		if err != nil {
			continue
		}

		tcb := tcb.GenerateTcb(state, class, root, content)
		tcbBlock := tcb.ToString()

		analyses = analyseCfg(cfg.BuildGraphFromContent(tcbBlock), analyses, buildPugAnalysis(tcb))
	}

	return analyses
}

func analyseCfg(cfgState *cfg.State, analyses []Analysis, buildAnalysis func(string, *sitter.Node, int) *Analysis) []Analysis {
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

			a := buildAnalysis(message, node, AnalysisSeverity.Error)
			if a != nil {
				analyses = append(analyses, *a)
			}
		}
	}

	return analyses
}
