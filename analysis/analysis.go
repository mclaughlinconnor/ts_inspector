package analysis

import (
	"fmt"
	"ts_inspector/analysis/cfg"
	"ts_inspector/config"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type analyser struct {
	exec      func(state *parser.State, file *parser.File) ([]Analysis, error)
	expensive bool
	name      string // mostly for logging
}

var Analysers = []analyser{}

func registerAnalyser(analyser analyser) {
	Analysers = append(Analysers, analyser)
}

func Analyse(state *parser.State, file *parser.File, runExpensive bool) []Analysis {
	analyses := []Analysis{}

	var errors = []error{}

	for _, analyser := range Analysers {
		if !runExpensive && analyser.expensive {
			continue
		}

		a, err := analyser.exec(state, file)
		if err != nil {
			errors = append(errors, fmt.Errorf("%v: %w", analyser.name, err))
		}

		analyses = append(analyses, a...)
	}

	analyses = append(analyses, errorsToAnalyses(errors)...)

	return analyses
}

func analyseClasses(file *parser.File, analyse func(class *parser.Class) ([]Analysis, error)) []Analysis {
	analyses := []Analysis{}

	var errors = []error{}

	for _, class := range file.Snapshot().Classes {
		if file.Snapshot().URI != class.Snapshot().File.Snapshot().URI {
			continue
		}

		a, err := analyse(class)
		if err != nil {
			errors = append(errors, fmt.Errorf("anonymous: %w", err))
		}

		analyses = append(analyses, a...)
	}

	analyses = append(analyses, errorsToAnalyses(errors)...)

	return analyses
}

func errorToAnalyses(err error) Analysis {
	return newAnalysis("analysisError", utils.ZeroRange(), AnalysisSeverity.Error, err.Error(), nil)
}

func errorsToAnalyses(errors []error) []Analysis {
	if len(errors) == 0 {
		return []Analysis{}
	}

	analyses := []Analysis{}
	for _, err := range errors {
		analyses = append(analyses, errorToAnalyses(err))
	}

	return analyses
}

func newAnalysisHighlightName(problemNode *sitter.Node, class *parser.Class, severity int, code string, message string) Analysis {
	var highlightNode *sitter.Node

	nameNode := problemNode.ChildByFieldName("name")
	if nameNode != nil {
		highlightNode = nameNode
	} else {
		highlightNode = problemNode
	}

	startByte := highlightNode.StartByte()
	endByte := highlightNode.EndByte()

	startByte += class.Snapshot().Node.StartByte()
	endByte += class.Snapshot().Node.StartByte()

	content := class.Snapshot().File.Snapshot().Content

	startPosition := utils.GetPositionForOffset(content, startByte)
	endPosition := utils.GetPositionForOffset(content, endByte)

	return newAnalysis(code, utils.Range{Start: startPosition, End: endPosition}, severity, message, nil)
}

func newAnalysis(code string, highlightRange utils.Range, severity int, message string, relatedInformation *[]RelatedInformation) Analysis {
	var ri []RelatedInformation
	if relatedInformation == nil {
		ri = []RelatedInformation{}
	} else {
		ri = *relatedInformation
	}

	return Analysis{code, message, highlightRange, ri, severity, "ts_inspector"}
}

func newAnalysisFromFileContent(content string, code string, node *sitter.Node, severity int, message string, relatedInformation *[]RelatedInformation) Analysis {
	var ri []RelatedInformation
	if relatedInformation == nil {
		ri = []RelatedInformation{}
	} else {
		ri = *relatedInformation
	}

	startPosition := utils.GetPositionForOffset(content, node.StartByte())
	endPosition := utils.GetPositionForOffset(content, node.EndByte())

	rrange := utils.Range{Start: startPosition, End: endPosition}

	return Analysis{code, message, rrange, ri, severity, "ts_inspector"}
}

func newAnalysisFromFileNode(file *parser.File, code string, node *sitter.Node, severity int, message string, relatedInformation *[]RelatedInformation) Analysis {
	content := file.Snapshot().Content

	return newAnalysisFromFileContent(content, code, node, severity, message, relatedInformation)
}

func InitAnalysers() {
	registerAnalyser(analyser{exec: angularManyDecorators, expensive: false, name: "angularManyDecorators"})
	registerAnalyser(analyser{exec: angularMethodNoImplements, expensive: false, name: "angularMethodNoImplements"})
	registerAnalyser(analyser{exec: asyncAngular, expensive: false, name: "asyncAngular"})
	registerAnalyser(analyser{exec: cfgUnreachableBlock, expensive: true, name: "cfgUnreachableBlock"})
	registerAnalyser(analyser{exec: constructorOnlyProperty, expensive: false, name: "constructorOnlyProperty"})
	registerAnalyser(analyser{exec: getterUsedInTemplate, expensive: false, name: "getterUsedInTemplate"})
	registerAnalyser(analyser{exec: illegalDeclaringModule, expensive: false, name: "illegalDeclaringModule"})
	registerAnalyser(analyser{exec: nonPublicAngular, expensive: false, name: "nonPublicAngular"})
	registerAnalyser(analyser{exec: recursiveTemplate, expensive: false, name: "recursiveTemplate"})
	registerAnalyser(analyser{exec: structuralDirectiveUnfoundKeyExprKey, expensive: true, name: "structuralDirectiveUnfoundKeyExprKey"})
	registerAnalyser(analyser{exec: unnecessaryPublic, expensive: false, name: "unnecessaryPublic"})
	registerAnalyser(analyser{exec: unusedAngular, expensive: false, name: "unusedAngular"})

	if config.GetConfig().Debug {
		registerAnalyser(analyser{exec: debug, expensive: true, name: "debug"})
	}

	if config.GetConfig().TsGo.Enable {
		registerAnalyser(analyser{exec: typescript, expensive: true, name: "typescript"})
	}

	cfg.InitBuilder()
}
