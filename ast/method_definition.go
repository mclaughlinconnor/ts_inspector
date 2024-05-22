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
			result.Node = node[0]
		}

		semi := captures["semi"]
		if semi != nil {
			result.Node = node[0]
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
		return int(a.Node.StartByte()) - int(b.Node.StartByte())
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
		insertPosition := utils.PositionFromPoint((*methodResults)[insertionIndex].Node.StartPoint())
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

type MethodDefinitionParseResult struct {
	Name string
	Node *sitter.Node
	Type string
}

type MethodDefinitions map[string]MethodDefinitionParseResult
