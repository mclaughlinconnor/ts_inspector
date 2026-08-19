package analysis

import (
	"fmt"
	"slices"
	"strings"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"
)

func structuralDirectiveUnfoundKeyExprKey(state *parser.State, file *parser.File) []Analysis {
	analyses := []Analysis{}

	if file.Snapshot().Filetype != "pug" {
		return analyses
	}

	if len(file.Snapshot().Classes) == 0 {
		return analyses
	}

	throwGeneric := func(err error) []Analysis {
		analyses = append(analyses, newAnalysis("structuralDirectiveUnfoundKeyExprKey", utils.ZeroRange(), AnalysisSeverity.Error, err.Error(), nil))
		return analyses
	}

	strContent := file.Snapshot().Content
	byteContent := []byte(strContent)

	root, err := utils.ParseText(byteContent, utils.Pug)
	if err != nil {
		return throwGeneric(err)
	}

	ast, err := tcb.Parse(root, byteContent, &tcb.Tcb{}), nil
	if err != nil {
		return throwGeneric(err)
	}

	analyseClass := func(class *parser.Class, attribute *tcb.Attribute) {
		valueShv, err := attribute.GetShv()
		if err != nil {
			analyses = append(analyses, newAnalysisFromFileNode(file, "structuralDirectiveUnfoundKeyExprKey", attribute.ValueNode, AnalysisSeverity.Error, err.Error(), nil))
			return
		}

		if !class.HasComponent() {
			return
		}

		foundMatch := false

		for _, thing := range class.Snapshot().Angular.Component.GetAvailableThings(state) {
			if !thing.HasDirective() {
				continue
			}

			for _, selector := range thing.GetSelectors() {
				matchesTag, _ := attribute.Tag.MatchesSelector(selector)
				if !matchesTag {
					continue
				}

				matchesAttribute, _ := attribute.MatchesSelector(selector, true, true)
				if !matchesAttribute {
					continue
				}

				foundMatch = true

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

		if !foundMatch {
			shv, err := attribute.GetShv()
			if err != nil {
				return
			}

			keyExprs := strings.Builder{}
			for _, expr := range shv.Statements.Elements {
				if !expr.HasKeyExp() {
					continue
				}

				keyExprs.WriteString("[")
				keyExprs.WriteString(expr.KeyExp.GetFullName(shv))
				keyExprs.WriteString("]")
			}

			message := fmt.Sprintf("[%v]%v is not selected by any directive", attribute.GetStrippedName(), keyExprs.String())
			analysis := newAnalysisFromFileNode(file, "structuralDirectiveUnfoundKeyExprKey", attribute.NameNode, AnalysisSeverity.Error, message, nil)
			analyses = append(analyses, analysis)
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
			break
		}

		analyse(attribute)
	}

	for _, c := range node.GetChildren().Elements {
		visit(c, analyses, analyse)
	}
}
