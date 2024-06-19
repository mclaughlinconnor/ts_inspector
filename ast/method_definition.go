package ast

import (
	"cmp"
	"slices"
	walktypescript "ts_inspector/ast/walk_typescript"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

// field public static = 300 + 2 + 10
// field protected static = 300 + 1 + 10
// field private static = 300 + 0 + 10
// field public = 300 + 2 + 0
// field protected = 300 + 1 + 0
// field private = 300 + 0 + 0
// constructor = 200
// property public = 100 + 2 + 0
// property protected = 100 + 1 + 0
// property private = 100 + 0 + 0
// method public static = 0 + 2 + 10
// method protected static = 0 + 1 + 10
// method private static = 0 + 0 + 10
// method public = 0 + 2
// method protected = 0 + 1
// method private = 0 + 0

/*
  private fifteen(): number {}
  private get nine(): number {}
  private six : number;
  private static three : number;
  private static twelve(): number {}
  protected five : number;
  protected fourteen(): number {}
  protected get eight(): number {}
  protected static eleven(): number {}
  protected static two : number;
  public constructor() {}
  public four : number;
  public get seven(): number {}
  public static one : number;
  public static ten(): number {}
  public thirteen(): number {}
*/

func calculateSortScore(node *sitter.Node, content []byte) int {
	name := node.ChildByFieldName("name")
	if name.Content(content) == "constructor" {
		return 200
	}

	score := 0

	if node.Type() == "public_field_definition" {
		score = score + 300
	}

	child := node.Child(0)
	for child != nil {
		if child.Type() == "accessibility_modifier" {
			modifier := child.Content(content)
			if modifier == "public" {
				score = score + 2
			} else if modifier == "protected" {
				score = score + 1
			} else if modifier == "private" {
				score = score + 0
			}
		}

		if child.Type() == "get" || child.Type() == "set" {
			score = score + 100
		}

		if child.Type() == "static" {
			score = score + 10
		}

		child = child.NextSibling()
	}

	return score
}

func ExtractDefinitions(content []byte) []MethodDefinitionParseResult {
	node, _ := utils.ParseText(content, utils.TypeScript, nil, func(root *sitter.Node, content []byte, state *sitter.Node) (*sitter.Node, error) {
		state = root
		return state, nil
	})

	funcMap := walktypescript.NewVisitorFuncsMap[[]MethodDefinitionParseResult]()

	methodHandler := func(node *sitter.Node, state []MethodDefinitionParseResult, indexInParent int) []MethodDefinitionParseResult {
		result := MethodDefinitionParseResult{}

		result.Range = utils.Range{Start: utils.PositionFromPoint(node.StartPoint()), End: utils.PositionFromPoint(node.EndPoint())}
		result.Type = node.Type()

		possibleSemiOrComment := node.NextSibling()
		for {
			if possibleSemiOrComment.Type() == ";" || possibleSemiOrComment.Type() == "comment" {
				point := utils.PositionFromPoint(possibleSemiOrComment.EndPoint())
				if result.Range.End.Line == point.Line {
					result.Range.End = utils.PositionFromPoint(possibleSemiOrComment.EndPoint())
					possibleSemiOrComment = possibleSemiOrComment.NextSibling()
					continue
				}

			}

			break
		}

		prev := node.PrevSibling()
		for prev.Type() == "decorator" || prev.Type() == "comment" {
			pprev := prev.PrevSibling()

			if prev.StartPoint().Row == pprev.StartPoint().Row {
				break // A comment or decorator on the same line as something else
			}

			result.Range.Start = utils.PositionFromPoint(prev.StartPoint())
			prev = pprev
		}

		name := node.ChildByFieldName("name")
		nameContent := name.Content(content)
		result.Name = nameContent

		result.Score = calculateSortScore(node, content)

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

func AddToMethodDefinition(methodResults *[]MethodDefinitionParseResult, classBodyNode *sitter.Node, toAdd string, name string, score int) utils.TextEdits {
	fullySorted := slices.IsSortedFunc(*methodResults, func(a MethodDefinitionParseResult, b MethodDefinitionParseResult) int {
		return cmp.Or(
			cmp.Compare(b.Score, a.Score),
			cmp.Compare(a.Name, b.Name),
		)
	})

	typesSorted := slices.IsSortedFunc(*methodResults, func(a MethodDefinitionParseResult, b MethodDefinitionParseResult) int {
		return cmp.Compare(b.Score, a.Score)
	})

	insertionIndex := -1
	if !fullySorted && !typesSorted {
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
	} else if !fullySorted && typesSorted {
		for index, result := range *methodResults {
			if score <= result.Score {
				insertionIndex = index
				break
			}
		}
	} else if fullySorted {
		for index, result := range *methodResults {
			if score <= result.Score && cmp.Compare(name, result.Name) == 1 {
				insertionIndex = index
			}
		}
	}

	if insertionIndex != -1 {
		insertPosition := (*methodResults)[insertionIndex].Range.End
		editRange := utils.Range{Start: insertPosition, End: insertPosition}

		insertionText := "\n\n" + toAdd

		return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: insertionText}}
	}

	if insertionIndex == -1 {
		if len(*methodResults) == 0 {
			insertionText := "{\n" + toAdd + "\n}"
			editRange := utils.Range{Start: utils.PositionFromPoint(classBodyNode.StartPoint()), End: utils.PositionFromPoint(classBodyNode.EndPoint())}
			return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: insertionText}}
		} else {
			insertPosition := (*methodResults)[0].Range.Start
			insertPosition.Character = 0
			editRange := utils.Range{Start: insertPosition, End: insertPosition}
			insertionText := toAdd + "\n\n"

			return utils.TextEdits{utils.TextEdit{Range: editRange, NewText: insertionText}}
		}
	}

	return nil
}

// Should handle type methoddefinitions
func AddMethodDefinitionToFile(content []byte, toAdd string, name string, score int) (utils.TextEdits, error) {
	edits := utils.TextEdits{}

	definitionResults := ExtractDefinitions(content)

	definitionResult, err := FindMethodDefinition(&definitionResults, toAdd)
	if err != nil || definitionResult != nil {
		return edits, err
	}

	classBodyNode := FindClassBody(content)
	if classBodyNode == nil {
		return edits, nil
	}
	methoddefinitionEdits := AddToMethodDefinition(&definitionResults, classBodyNode, toAdd, name, score)

	return methoddefinitionEdits, nil

}

type MethodDefinitionParseResult struct {
	Name  string
	Range utils.Range
	Score int
	Text  string
	Type  string
}

type MethodDefinitions map[string]MethodDefinitionParseResult
