package analysis

import (
	"fmt"
	"ts_inspector/analysis/cfg"
	"ts_inspector/config"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

const unreachableCode = "unreachable"

func cfgUnreachableBlock(state *parser.State, file *parser.File) ([]Analysis, error) {
	analyses := []Analysis{}

	if file.Snapshot().Filetype == "typescript" {
		cfg, err := cfg.BuildGraphFromFile(file)
		if err != nil {
			return analyses, err
		}

		return analyseCfg(file.Snapshot().Content, cfg, analyses, func(m string, n *sitter.Node, s int) *Analysis {
			a := newAnalysisFromFileNode(file, unreachableCode, n, s, m, nil)
			return &a
		}), nil
	}

	if file.Snapshot().Filetype != "pug" {
		return analyses, nil
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

		tcb, err := tcb.GenerateTcb(state, class, root, content)
		if err != nil {
			return nil, err
		}

		tcbBlock := tcb.ToString()

		cfg, err := cfg.BuildGraphFromContent(tcbBlock)
		if err != nil {
			return analyses, err
		}

		analyses = analyseCfg(tcbBlock, cfg, analyses, buildPugAnalysis(tcb))
	}

	return analyses, nil
}

func analyseCfg(content string, cfgState *cfg.State, analyses []Analysis, buildAnalysis func(string, *sitter.Node, int) *Analysis) []Analysis {
	for _, cfg := range cfgState.AllCfg {
		analyses = analyseComplexity(analyses, content, cfg)

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

func analyseComplexity(analyses []Analysis, content string, cfg *cfg.FunctionCFG) []Analysis {
	complexity := cfg.CalculateCyclomaticComplexity()
	if complexity <= 10 {
		return analyses
	}

	var level string
	var severity int

	if complexity <= 20 {
		level = "Moderate"
		severity = AnalysisSeverity.Warning
	} else if complexity <= 50 {
		level = "High"
		severity = AnalysisSeverity.Error
	} else {
		level = "Very high"
		severity = AnalysisSeverity.Error
	}

	message := fmt.Sprintf("%v complexity: %v", level, cfg.CalculateCyclomaticComplexity())

	analysis := newAnalysisFromFileContent(content, "complexity", cfg.Node, severity, message, nil)
	analysis.Range.End = analysis.Range.Start
	analyses = append(analyses, analysis)

	return analyses
}
