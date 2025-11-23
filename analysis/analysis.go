package analysis

import (
	"fmt"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

var enableDebug = false

type analyser = func(file *parser.File) []Analysis

var Analysers = []analyser{}

func registerAnalyser(analyser analyser) {
	Analysers = append(Analysers, analyser)
}

func Analyse(file *parser.File) []Analysis {
	analyses := []Analysis{}

	for _, analyser := range Analysers {
		analyses = append(analyses, analyser(file)...)
	}

	return analyses
}

func analyseClasses(file *parser.File, analyse func(class parser.Class) []Analysis) []Analysis {
	analyses := []Analysis{}

	for _, class := range file.Snapshot().Classes {
		if file.Snapshot().URI != class.File.Snapshot().URI {
			continue
		}

		analyses = append(analyses, analyse(*class)...)
	}

	return analyses
}

func newAnalysisHighlightName(problemNode *sitter.Node, class parser.Class, severity int, code string, message string) Analysis {
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

	startByte += class.Node.StartByte()
	endByte += class.Node.StartByte()

	content := class.File.Snapshot().Content

	startPosition := parser.GetPositionForOffset(content, startByte)
	endPosition := parser.GetPositionForOffset(content, endByte)

	return newAnalysis(code, utils.Range{Start: startPosition, End: endPosition}, severity, message)
}

func newAnalysis(code string, highlightRange utils.Range, severity int, message string) Analysis {
	return Analysis{code, message, highlightRange, severity, "ts_inspector"}
}

func InitAnalysers() {
	registerAnalyser(angularManyDecorators)
	registerAnalyser(angularMethodNoImplements)
	registerAnalyser(asyncAngular)
	registerAnalyser(constructorOnlyProperty)
	registerAnalyser(getterUsedInTemplate)
	registerAnalyser(illegalDeclaringModule)
	registerAnalyser(nonPublicAngular)
	registerAnalyser(recursiveTemplate)
	registerAnalyser(unnecessaryPublic)
	registerAnalyser(unusedAngular)

	if enableDebug {
		registerAnalyser(debug)
	}
}
