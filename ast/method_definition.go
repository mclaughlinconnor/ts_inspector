package ast

import (
	walktypescript "ts_inspector/ast/walk_typescript"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractMethodDefinitions(content []byte) []MethodDefinitionParseResult {
	node, _ := utils.ParseText(content, utils.TypeScript, nil, func(root *sitter.Node, content []byte, state *sitter.Node) (*sitter.Node, error) {
		state = root
		return state, nil
	})

	funcMap := walktypescript.NewVisitorFuncsMap[[]MethodDefinitionParseResult]()

	methodHandler := func(node *sitter.Node, state []MethodDefinitionParseResult, indexInParent int) []MethodDefinitionParseResult {
		result := MethodDefinitionParseResult{}

		result.Range = utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}
		result.Type = node.Type()

		possibleSemi := node.NextSibling()
		if possibleSemi.Type() == ";" {
			result.Range.End = utils.PositionFromPoint(node.EndPoint())
		}

		prev := node.PrevSibling()
		for prev.Type() == "decorator" || prev.Type() == "comment" {
			result.Range.Start = utils.PositionFromPoint(prev.StartPoint())
			prev = prev.PrevSibling()
		}

		name := node.ChildByFieldName("name")
		result.Name = name.Content(content)

		return append(state, result)
	}

	funcMap["public_field_definition"] = methodHandler
	funcMap["method_definition"] = methodHandler
	funcMap["method_signature"] = methodHandler
	funcMap["abstract_method_signature"] = methodHandler

	return walktypescript.Walk(node, []MethodDefinitionParseResult{}, funcMap)
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

	definitionResults := ExtractMethodDefinitions(content)
	definitionResult, err := FindMethodDefinition(&definitionResults, toAdd)
	if err != nil || definitionResult != nil {
		return edits, err
	}

	classBodyNode := FindClassBody(content)
	methoddefinitionEdits := AddToMethodDefinition(&definitionResults, classBodyNode, toAdd, name)

	return methoddefinitionEdits, nil

}

type MethodDefinitionParseResult struct {
	Name  string
	Range utils.Range
	Type  string
}

type MethodDefinitions map[string]MethodDefinitionParseResult
