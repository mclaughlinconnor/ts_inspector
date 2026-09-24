package analysis

import (
	"fmt"
	"ts_inspector/ast/manipulation"
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

const unreachableCode = "unreachable"

func cfgUnreachableBlock(state *parser.State, file *parser.File) ([]interfaces.Analysis, error) {
	analyses := []interfaces.Analysis{}

	if file.Snapshot().Filetype == "typescript" {
		cfg, err := file.Snapshot().Ast.GetCfg()
		if err != nil {
			return analyses, err
		}

		return analyseCfg(file.Snapshot().Content, cfg, analyses, false, func(m string, n manipulation.AstManipulationNode, s int) *interfaces.Analysis {
			a := interfaces.Analysis{Code: unreachableCode, Message: m, Range: n.GetRange(), RelatedInformation: nil, Severity: s, Source: "ts_inspector"}
			return &a
		}), nil
	}

	if file.Snapshot().Filetype != "pug" {
		return analyses, nil
	}

	content := []byte(file.Snapshot().Content)

	buildPugAnalysis := func(tcbBlock *tcb.Statement) func(string, manipulation.AstManipulationNode, int) *interfaces.Analysis {
		return func(message string, node manipulation.AstManipulationNode, severity int) *interfaces.Analysis {
			r := tcbBlock.TsOffsetToRange(file.Snapshot().Content, int(node.GetStartOffset()), int(node.GetEndOffset()), config.GetConfig().Debug)
			if r == nil {
				return nil
			}

			a := interfaces.NewAnalysis(unreachableCode, *r, severity, message, nil)

			return &a
		}
	}

	for _, class := range file.Snapshot().Classes {
		root, err := utils.ParseText([]byte(content), utils.Pug)
		if err != nil {
			return nil, err
		}

		tcb, err := tcb.GenerateTcb(state, class, root, content)
		if err != nil {
			return nil, err
		}

		tcbBlock := tcb.ToString()

		ast, err := manipulation.BuildAst(tcbBlock)
		if err != nil {
			return analyses, err
		}

		cfg, err := ast.GetCfg()
		if err != nil {
			return analyses, err
		}

		analyses = analyseCfg(tcbBlock, cfg, analyses, true, buildPugAnalysis(tcb))
	}

	return analyses, nil
}

func analyseCfg(content string, cfgState *manipulation.Cfg, analyses []interfaces.Analysis, skipComplexity bool, buildAnalysis func(string, manipulation.AstManipulationNode, int) *interfaces.Analysis) []interfaces.Analysis {
	for _, cfg := range cfgState.GetAllFunctionCfg() {
		if !skipComplexity {
			analyses = analyseComplexity(analyses, content, cfg)
		}

		for _, block := range cfg.GetBlocks() {
			if len(block.Before) != 0 || cfg.Start == block {
				continue
			}

			message := "Code is unreachable"

			var node manipulation.AstManipulationNode
			if block.Node != nil {
				node = block.Node
			} else if len(block.Instructions) != 0 {
				node = block.Instructions[0].Node
			}

			if node == nil {
				continue
			}

			a := buildAnalysis(message, node, interfaces.AnalysisSeverity.Error)
			if a != nil {
				analyses = append(analyses, *a)
			}
		}
	}

	return analyses
}

func analyseComplexity(analyses []interfaces.Analysis, content string, cfg *manipulation.FunctionCfg) []interfaces.Analysis {
	complexity := cfg.CalculateCyclomaticComplexity()
	if complexity <= 10 {
		return analyses
	}

	var level string
	var severity int

	if complexity <= 20 {
		level = "Moderate"
		severity = interfaces.AnalysisSeverity.Warning
	} else if complexity <= 50 {
		level = "High"
		severity = interfaces.AnalysisSeverity.Error
	} else {
		level = "Very high"
		severity = interfaces.AnalysisSeverity.Error
	}

	message := fmt.Sprintf("%v complexity: %v", level, cfg.CalculateCyclomaticComplexity())

	analysis := interfaces.Analysis{Code: "complexity", Message: message, Range: cfg.Node.GetRange(), RelatedInformation: nil, Severity: severity, Source: "ts_inspector"}

	analysis.Range.End = analysis.Range.Start
	analyses = append(analyses, analysis)

	return analyses
}
