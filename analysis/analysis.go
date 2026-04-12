package analysis

import (
	"ts_inspector/analysis/cfg"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type analyser struct {
	exec      func(state *parser.State, file *parser.File) []Analysis
	expensive bool
}

var Analysers = []analyser{}

func registerAnalyser(analyser analyser) {
	Analysers = append(Analysers, analyser)
}

func Analyse(state *parser.State, file *parser.File, runExpensive bool) []Analysis {
	analyses := []Analysis{}

	for _, analyser := range Analysers {
		if runExpensive || !analyser.expensive {
			analyses = append(analyses, analyser.exec(state, file)...)
		}
	}

	return analyses
}

func analyseClasses(file *parser.File, analyse func(class *parser.Class) []Analysis) []Analysis {
	analyses := []Analysis{}

	for _, class := range file.Snapshot().Classes {
		if file.Snapshot().URI != class.Snapshot().File.Snapshot().URI {
			continue
		}

		analyses = append(analyses, analyse(class)...)
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

func InitAnalysers() {
	registerAnalyser(analyser{exec: angularManyDecorators, expensive: false})
	registerAnalyser(analyser{exec: angularMethodNoImplements, expensive: false})
	registerAnalyser(analyser{exec: asyncAngular, expensive: false})
	registerAnalyser(analyser{exec: cfgUnreachableBlock, expensive: true})
	registerAnalyser(analyser{exec: constructorOnlyProperty, expensive: false})
	registerAnalyser(analyser{exec: getterUsedInTemplate, expensive: false})
	registerAnalyser(analyser{exec: illegalDeclaringModule, expensive: false})
	registerAnalyser(analyser{exec: nonPublicAngular, expensive: false})
	registerAnalyser(analyser{exec: recursiveTemplate, expensive: false})
	registerAnalyser(analyser{exec: unnecessaryPublic, expensive: false})
	registerAnalyser(analyser{exec: unusedAngular, expensive: false})

	if utils.Debug {
		registerAnalyser(analyser{exec: debug, expensive: true})
	}

	if utils.TsGo {
		registerAnalyser(analyser{exec: typescript, expensive: true})
	}

	cfg.InitBuilder()
}
