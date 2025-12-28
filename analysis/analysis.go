package analysis

import (
	"fmt"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type analyser struct {
	exec      func(file *parser.File) []Analysis
	expensive bool
}

var Analysers = []analyser{}

func registerAnalyser(analyser analyser) {
	Analysers = append(Analysers, analyser)
}

func Analyse(file *parser.File, runExpensive bool) []Analysis {
	analyses := []Analysis{}

	for _, analyser := range Analysers {
		if !runExpensive || (runExpensive && analyser.expensive) {
			analyses = append(analyses, analyser.exec(file)...)
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
		fmt.Println(nameNode == nil)
		highlightNode = nameNode
	} else {
		fmt.Println(nameNode == nil)
		highlightNode = problemNode
	}

	startByte := highlightNode.StartByte()
	endByte := highlightNode.EndByte()

	startByte += class.Snapshot().Node.StartByte()
	endByte += class.Snapshot().Node.StartByte()

	content := class.Snapshot().File.Snapshot().Content

	startPosition := parser.GetPositionForOffset(content, startByte)
	endPosition := parser.GetPositionForOffset(content, endByte)

	return newAnalysis(code, utils.Range{Start: startPosition, End: endPosition}, severity, message)
}

func newAnalysis(code string, highlightRange utils.Range, severity int, message string) Analysis {
	return Analysis{code, message, highlightRange, severity, "ts_inspector"}
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
}
