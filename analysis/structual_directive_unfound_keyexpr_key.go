package analysis

import (
	"fmt"
	"slices"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func structuralDirectiveUnfoundKeyExprKey(state *parser.State, file *parser.File) []Analysis {
	analyses := []Analysis{}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	if len(file.Snapshot().Classes) == 0 {
		return analyses
	}

	strContent := file.Snapshot().Content
	byteContent := []byte(strContent)
	ast, err := utils.ParseText(byteContent, utils.Pug, nil, func(root *sitter.Node, _ []byte, _ *tcb.Ast) (*tcb.Ast, error) {
		return tcb.Parse(root, byteContent, &tcb.Tcb{}), nil
	})

	if err != nil {
		analyses = append(analyses, newAnalysis("structuralDirectiveUnfoundKeyExprKey", utils.ZeroRange(), AnalysisSeverity.Error, err.Error(), nil))
		return analyses
	}

	analyseClass := func(class *parser.Class, attribute *tcb.Attribute) {
		valueShv, err := attribute.GetShv()
		if err != nil {
			analyses = append(analyses, newAnalysis("structuralDirectiveUnfoundKeyExprKey", utils.ZeroRange(), AnalysisSeverity.Error, err.Error(), nil))
			return
		}

		if !class.HasComponent() {
			return
		}

		for _, thing := range class.Snapshot().Angular.Component.GetAvailableThings(state) {
			for _, selector := range thing.GetSelectors() {
				if !attribute.Tag.MatchesSelector(selector) {
					continue
				}

				definitions := thing.GetAllDefinitions()

				for _, statement := range valueShv.Statements.Elements {
					if !statement.HasKeyExp() {
						continue
					}

					keyExp := statement.KeyExp

					validKeyExpKey := slices.ContainsFunc(definitions, func(d parser.ClassedDefinition) bool {
						return keyExp.Matches(valueShv, d.GetInputName())
					})

					if validKeyExpKey {
						continue
					}

					startOffset := attribute.ValueNode.StartByte() + uint32(keyExp.KeyOffset)
					endOffset := startOffset + uint32(len(keyExp.Key))

					startPosition := utils.GetPositionForOffset(strContent, startOffset)
					endPosition := utils.GetPositionForOffset(strContent, endOffset)

					rrange := utils.Range{Start: startPosition, End: endPosition}

					// TODO: Add a "did you mean ..."
					message := fmt.Sprintf("Key %v doesn't exist on %v.", keyExp.GetFullName(valueShv), thing.Snapshot().Name)

					analysis := newAnalysis("structuralDirectiveUnfoundKeyExprKey", rrange, AnalysisSeverity.Error, message, nil)
					analyses = append(analyses, analysis)
				}
			}
		}
	}

	analyse := func(attribute *tcb.Attribute) {
		for _, class := range file.Snapshot().Classes {
			analyseClass(class, attribute)
		}
	}

	visit(ast.Root, &analyses, analyse)

	return analyses
}

func visit(node *tcb.Node, analyses *[]Analysis, analyse func(*tcb.Attribute)) {
	if node.Kind != tcb.KindTag {
		for _, c := range node.GetChildren().Elements {
			visit(c, analyses, analyse)
		}

		return
	}

	for _, attributeNode := range node.Tag.Attributes.Elements {
		attribute := attributeNode.Attribute
		if !attribute.IsStructuralInput() {
			return
		}

		analyse(attribute)
	}

	for _, c := range node.GetChildren().Elements {
		visit(c, analyses, analyse)
	}
}
