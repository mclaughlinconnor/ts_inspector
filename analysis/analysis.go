package analysis

import (
	"fmt"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func Analyse(file parser.File) []Analysis {
	analyses := []Analysis{}

	for _, class := range file.Classes {
		analyses = append(analyses, analyseClass(*class)...)
	}

	return analyses
}

func analyseClass(class parser.Class) []Analysis {
	analyses := []Analysis{}

	getters := class.GetGetters()
	definitions := class.Definitions

	if class.AngularTemplateFile == nil {
		// No analysis for files that are not angular controllers
		return analyses
	}

	for _, definition := range getters {
		used := len(definition.Usages) != 0
		if used && definition.UsageAccess == parser.TemplateAccess {
			message := fmt.Sprintf("Getter used in template: %s", definition.Name)
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Hint, message))
		}
	}

	for _, definition := range definitions {
		definitionIsPublic := definition.AccessModifier == parser.PublicAccessibility
		definitionIsLocalParam := definition.AccessModifier == parser.NoAccessibility
		used := len(definition.Usages) != 0

		if used && definition.IsConstructorParam() && definition.UsageAccess == parser.ConstructorAccess {
			message := fmt.Sprintf("Variable only used in constructor: %s", definition.Name)
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, message))
		}

		var hasAngularDecorator bool = false
		for _, decorator := range definition.Decorators {
			hasAngularDecorator = hasAngularDecorator || decorator.IsAngular
		}

		if definitionIsPublic && !hasAngularDecorator && !definition.Static && !definition.IsAngularMethod && !definition.Override {
			if !used {
				message := fmt.Sprintf("Unused public variable: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, message))
			} else if definition.UsageAccess != parser.TemplateAccess {
				message := fmt.Sprintf("Needlessly public variable: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, message))
			}
		}

		if hasAngularDecorator && len(definition.Usages) == 0 && !definitionIsLocalParam {
			if definition.Override {
				message := fmt.Sprintf("Angular property never used in this component: %s. Check the parent class.", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Hint, message))
			} else {
				message := fmt.Sprintf("Angular property never used in this component: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, message))
			}
		}

		if hasAngularDecorator && !definitionIsPublic && !definitionIsLocalParam {
			message := fmt.Sprintf("Angular property should be public: %s", definition.Name)
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, message))
		}

		if definition.IsAngularMethod && definition.Async {
			message := "Angular method must not be async"
			analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Error, message))
		}
	}

	return analyses
}

func newAnalysisHighlightName(problemNode *sitter.Node, class parser.Class, severity int, message string) Analysis {
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

	startPosition := parser.GetPositionForOffset(class.File.Content, startByte)
	endPosition := parser.GetPositionForOffset(class.File.Content, endByte)

	return newAnalysis(utils.Range{Start: startPosition, End: endPosition}, severity, message)
}

func newAnalysis(highlightRange utils.Range, severity int, message string) Analysis {
	return Analysis{
		message,
		highlightRange,
		severity,
		"ts_inspector",
	}
}
