package actions

import (
	"fmt"
	"strings"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func MakeAsync(
	_ *utils.Writer,
	state *parser.State,
	file *parser.File,
	editRange utils.Range,
) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	if file.Snapshot().Filetype != "typescript" {
		return nil, nil, false, nil
	}

	var edits = utils.TextEdits{}

	content := []byte(file.Snapshot().Content)
	root := utils.ParseText(content, utils.TypeScript)

	cursor := root.Walk()
	cursor.GotoFirstChild() // go into (program)
	currentNode := cursor.Node()

	moved := false
	start, end := file.GetOffsetsForRange(editRange)

	for {
		if currentNode.StartByte() <= start && currentNode.EndByte() >= end { // if before start, keep going. If after end, stop (backtrack?)
			moved = cursor.GotoFirstChild()
			currentNode = cursor.Node()
		} else if currentNode.StartByte() > start {
			cursor.GotoParent() // reached a terminal node that is past the cursor, go back to the parent
			cursor.Node()
			break
		} else {
			moved = cursor.GotoNextSibling()
			currentNode = cursor.Node()
		}

		if !moved {
			break
		}
	}

	for {
		currentNode = cursor.Node()
		if currentNode.Kind() == "method_definition" || currentNode.Kind() == "function_declaration" || currentNode.Kind() == "arrow_function" {
			break
		}

		moved = cursor.GotoParent()

		// Don't have a method_definition in the heirarchy
		if !moved {
			return retEdits(nil, nil)
		}
	}

	cursor.GotoFirstChild()

	var postAsyncNode *sitter.Node
	hasAsync := false

	if currentNode.Kind() == "method_definition" {
		for cursor.GotoNextSibling() {
			fieldName := cursor.FieldName()
			currentNode = cursor.Node()
			if fieldName == "return_type" {
				break
			}

			if hasAsync {
				continue
			}

			fieldType := currentNode.Kind()
			if fieldType == "async" {
				hasAsync = true
				continue
			}

			if fieldType == "get" || fieldType == "set" || fieldType == "*" {
				postAsyncNode = cursor.Node()
			}

			if fieldName == "name" && postAsyncNode == nil {
				postAsyncNode = cursor.Node()
			}
		}

		if !hasAsync && postAsyncNode != nil {
			editRange := utils.Range{Start: utils.LspPositionFromTsPosition(postAsyncNode.StartPosition()), End: utils.LspPositionFromTsPosition(postAsyncNode.StartPosition())}
			edits = append(edits, utils.TextEdit{Range: editRange, NewText: "async "})
		}
	} else {
		nodeContent := currentNode.Utf8Text(content)
		if !strings.HasPrefix(nodeContent, "async ") {
			editRange := utils.Range{Start: utils.LspPositionFromTsPosition(currentNode.StartPosition()), End: utils.LspPositionFromTsPosition(currentNode.StartPosition())}
			edits = append(edits, utils.TextEdit{Range: editRange, NewText: "async "})
		}

		for cursor.GotoNextSibling() {
			fieldName := cursor.FieldName()
			if fieldName == "return_type" {
				break
			}
		}
	}

	if cursor.Node().Kind() == "type_annotation" {
		cursor.GotoFirstChild()  // ":"
		cursor.GotoNextSibling() // the type

		currentNode = cursor.Node()
		typeName := currentNode.Utf8Text(content)
		if !strings.HasPrefix(typeName, "Promise") { // Promise<Promise<void>> is almost certainly wrong
			editRange := utils.Range{Start: utils.LspPositionFromTsPosition(currentNode.StartPosition()), End: utils.LspPositionFromTsPosition(currentNode.EndPosition())}
			edits = append(edits, utils.TextEdit{Range: editRange, NewText: fmt.Sprintf("Promise<%s>", currentNode.Utf8Text(content))})
		}
	}

	return retEdits(&edits, nil)
}
