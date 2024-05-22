package ast

import (
	"slices"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractMethodDefinitions(content []byte) ([]MethodDefinitionParseResult, error) {
	result, err := utils.WithMatches(utils.QueryClassProperties, utils.TypeScript, content, []MethodDefinitionParseResult{}, func(captures utils.Captures, returnValue []MethodDefinitionParseResult) ([]MethodDefinitionParseResult, error) {
		result := MethodDefinitionParseResult{}

		name := captures["name"]
		if name != nil {
			result.Name = name[0].Content(content)
		}

		node := captures["node"]
		if node != nil {
			result.Range = utils.Range{Start: utils.PositionFromPoint(node[0].StartPoint()), End: utils.PositionFromPoint(node[0].EndPoint())}
			result.byteRange = byteRange{start: node[0].StartByte(), end: node[0].EndByte()}
		}

		semi := captures["semi"]
		if semi != nil {
			result.Range.End = utils.PositionFromPoint(node[0].EndPoint())
			result.byteRange.end = node[0].EndByte()
		}

		comments := captures["comment"]
		if comments != nil {
			var min uint32 = comments[0].StartByte()
			var minIndex int = 0
			for _, comment := range comments {
				if comment.StartByte() < min {
					min = comment.StartByte()
				}
			}

			result.Range.Start = utils.PositionFromPoint(comments[minIndex].StartPoint())
			result.byteRange.start = comments[minIndex].StartByte()
		}

		return append(returnValue, result), nil
	})

	return result, err
}

func FindClassBody(content []byte) *sitter.Node {
	node, _ := utils.WithMatches(utils.QueryClassBody, utils.TypeScript, content, nil, func(captures utils.Captures, returnValue *sitter.Node) (*sitter.Node, error) {
		if len(captures) == 1 && captures["body"] != nil {
			return captures["body"][0], nil
		}

		return nil, nil
	})

	return node
}

func FindMethodDefinition(methodDefinitionResults *[]MethodDefinitionParseResult, methodName string) (*MethodDefinitionParseResult, error) {
	for _, definition := range *methodDefinitionResults {
		if definition.Name == methodName {
			return &definition, nil
		}
	}

	return nil, nil
}

func AddToMethodDefinition(methodResults *[]MethodDefinitionParseResult, classBodyNode *sitter.Node, toAdd string, name string) utils.TextEdits {
	slices.SortFunc(*methodResults, func(a MethodDefinitionParseResult, b MethodDefinitionParseResult) int {
		return int(a.byteRange.start) - int(b.byteRange.start)
	})

	insertionIndex := -1
	for index, result := range *methodResults {
		if result.Name == name {
			return utils.TextEdits{}
		}

		if result.Type != "public_field_definition" && insertionIndex == -1 {
			if index == len(*methodResults)-1 {
				insertionIndex = len(*methodResults) - 1
			} else {
				insertionIndex = index
			}

			break
		}
	}

	if insertionIndex != -1 {
		insertPosition := (*methodResults)[insertionIndex].Range.Start
		insertPosition.Character = 0 // at the start of the line immediately following the node
		editRange := utils.Range{Start: insertPosition, End: insertPosition}

		insertionText := toAdd + "\n\n"

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: insertionText}}
	}

	if insertionIndex == -1 {
		insertionText := "{\n" + toAdd + "\n}"
		editRange := utils.Range{Start: utils.PositionFromPoint(classBodyNode.StartPoint()), End: utils.PositionFromPoint(classBodyNode.EndPoint())}
		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: insertionText}}
	}

	return nil
}

// Should handle type methoddefinitions
func AddMethodDefinitionToFile(content []byte, toAdd string, name string) (utils.TextEdits, error) {
	edits := utils.TextEdits{}

	definitionResults, err := ExtractMethodDefinitions(content)
	definitionResult, err := FindMethodDefinition(&definitionResults, toAdd)
	if err != nil || definitionResult != nil {
		return edits, err
	}

	classBodyNode := FindClassBody(content)
	methoddefinitionEdits := AddToMethodDefinition(&definitionResults, classBodyNode, toAdd, name)

	return methoddefinitionEdits, nil

}

type byteRange struct {
	start uint32
	end   uint32
}

type MethodDefinitionParseResult struct {
	Name      string
	Range     utils.Range
	byteRange byteRange
	Type      string
}

type MethodDefinitions map[string]MethodDefinitionParseResult
