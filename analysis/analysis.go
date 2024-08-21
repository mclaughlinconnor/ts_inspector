package analysis

import (
	"fmt"
	"ts_inspector/parser"

	sitter "github.com/smacker/go-tree-sitter"
)

func Analyse(file parser.File) []Analysis {
	analyses := []Analysis{}

	getters := file.GetGetters()
	definitions := file.Definitions

	if file.Template == "" {
		// No analysis for files that are not angular controllers
		return analyses
	}

	for _, definition := range getters {
		used := len(definition.Usages) != 0
		if used && definition.UsageAccess == parser.ForeignAccess {
			message := fmt.Sprintf("Getter used in template: %s", definition.Name)
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, AnalysisSeverity.Hint, message))
		}
	}

	for _, definition := range definitions {
		definitionIsPublic := definition.AccessModifier == parser.PublicAccessibility
		used := len(definition.Usages) != 0

		if used && definition.UsageAccess == parser.ConstructorAccess {
			message := fmt.Sprintf("Variable only used in constructor: %s", definition.Name)
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, AnalysisSeverity.Warning, message))
			continue
		}

		var hasAngularDecorator bool = false
		for _, decorator := range definition.Decorators {
			hasAngularDecorator = hasAngularDecorator || decorator.IsAngular
		}

		if definitionIsPublic && !hasAngularDecorator && !definition.Static && !definition.IsAngularMethod {
			if !used {
				message := fmt.Sprintf("Unused public variable: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, AnalysisSeverity.Warning, message))
			} else if definition.UsageAccess != parser.ForeignAccess {
				message := fmt.Sprintf("Needlessly public variable: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, AnalysisSeverity.Warning, message))
			}
		}
	}

	CurrentAnalysis[file.URI] = analyses

	return analyses
}

func newAnalysisHighlightName(problemNode *sitter.Node, severity int, message string) Analysis {
	var highlightNode *sitter.Node

	nameNode := problemNode.ChildByFieldName("name")
	if nameNode != nil {
    fmt.Println(nameNode == nil)
		highlightNode = nameNode
	} else {
    fmt.Println(nameNode == nil)
		highlightNode = problemNode
	}

	return newAnalysis(highlightNode, problemNode, severity, message)
}

func newAnalysis(highlightNode *sitter.Node, problemNode *sitter.Node, severity int, message string) Analysis {
	return Analysis{highlightNode, problemNode, severity, "ts_inspector", message}
}
