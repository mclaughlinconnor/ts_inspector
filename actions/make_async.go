package actions

import (
	"fmt"
	"strings"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func MakeAsync(
	state parser.State,
	file parser.File,
	editRange utils.Range,
) (actionEdits utils.TextEdits, allowed bool, err error) {
	if file.Filetype != "typescript" {
		return nil, false, nil
	}

	var edits = utils.TextEdits{}

	start, end := file.GetOffsetsForRange(editRange)

	action, err := utils.ParseFile(false, file.Content, utils.TypeScript, edits, func(root *sitter.Node, content []byte, edits utils.TextEdits) (utils.TextEdits, error) {
		cursor := sitter.NewTreeCursor(root)
		cursor.GoToFirstChild() // go into (program)
		currentNode := cursor.CurrentNode()

		moved := false

		for true {
			if currentNode.StartByte() <= start && currentNode.EndByte() >= end { // if before start, keep going. If after end, stop (backtrack?)
				moved = cursor.GoToFirstChild()
				currentNode = cursor.CurrentNode()
			} else if currentNode.StartByte() > start {
				cursor.GoToParent() // reached a terminal node that is past the cursor, go back to the parent
				currentNode = cursor.CurrentNode()
				break
			} else {
				moved = cursor.GoToNextSibling()
				currentNode = cursor.CurrentNode()
			}

			if !moved {
				break
			}
		}

		for true {
			currentNode = cursor.CurrentNode()
			if currentNode.Type() == "method_definition" || currentNode.Type() == "function_declaration" || currentNode.Type() == "arrow_function" {
				break
			}

			moved = cursor.GoToParent()

			// Don't have a method_definition in the heirarchy
			if !moved {
				return utils.TextEdits{}, nil
			}
		}

		cursor.GoToFirstChild()

		var postAsyncNode *sitter.Node
		hasAsync := false

		if currentNode.Type() == "method_definition" {
			for cursor.GoToNextSibling() {
				fieldName := cursor.CurrentFieldName()
				currentNode = cursor.CurrentNode()
				if fieldName == "return_type" {
					break
				}

				if hasAsync {
					continue
				}

				fieldType := currentNode.Type()
				if fieldType == "async" {
					hasAsync = true
					continue
				}

				if fieldType == "get" || fieldType == "set" || fieldType == "*" {
					postAsyncNode = cursor.CurrentNode()
				}

				if fieldName == "name" && postAsyncNode == nil {
					postAsyncNode = cursor.CurrentNode()
				}
			}

			if !hasAsync && postAsyncNode != nil {
				editRange := utils.Range{Start: utils.PositionFromPoint(postAsyncNode.StartPoint()), End: utils.PositionFromPoint(postAsyncNode.StartPoint())}
				edits = append(edits, utils.TextEdit{Range: editRange, NewText: "async "})
			}
		} else {
			nodeContent := currentNode.Content(content)
			if !strings.HasPrefix(nodeContent, "async ") {
				editRange := utils.Range{Start: utils.PositionFromPoint(currentNode.StartPoint()), End: utils.PositionFromPoint(currentNode.StartPoint())}
				edits = append(edits, utils.TextEdit{Range: editRange, NewText: "async "})
			}

			for cursor.GoToNextSibling() {
				fieldName := cursor.CurrentFieldName()
				currentNode = cursor.CurrentNode()
				if fieldName == "return_type" {
					break
				}
			}
		}

		if cursor.CurrentNode().Type() == "type_annotation" {
			cursor.GoToFirstChild()  // ":"
			cursor.GoToNextSibling() // the type

			currentNode = cursor.CurrentNode()
			typeName := currentNode.Content(content)
			if !strings.HasPrefix(typeName, "Promise") { // Promise<Promise<void>> is almost certainly wrong
				editRange := utils.Range{Start: utils.PositionFromPoint(currentNode.StartPoint()), End: utils.PositionFromPoint(currentNode.EndPoint())}
				edits = append(edits, utils.TextEdit{Range: editRange, NewText: fmt.Sprintf("Promise<%s>", currentNode.Content(content))})
			}
		}

		return edits, nil
	})

	return action, true, err
}
