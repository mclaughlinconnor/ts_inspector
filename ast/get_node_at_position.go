package ast

import (
	sitter "github.com/smacker/go-tree-sitter"
)

func GetNamedNodeAtPosition(root *sitter.Node, offset uint32) *sitter.Node {
	cursor := sitter.NewTreeCursor(root)
	node := cursor.CurrentNode()
	moved := false

	for true {
		if node.StartByte() <= offset && node.EndByte() >= offset { // if before startByte, keep going. If after endByte, stop (backtrack?)
			moved = cursor.GoToFirstChild()
			node = cursor.CurrentNode()
		} else if node.StartByte() > offset {
			cursor.GoToParent() // reached a terminal node that is past the cursor, go back to the parent
			node = cursor.CurrentNode()
			break
		} else {
			moved = cursor.GoToNextSibling()
			node = cursor.CurrentNode()
		}

		if !moved {
			break
		}
	}

	for true {
		node = cursor.CurrentNode()
		if node.IsNamed() {
			return node
		}

		moved = cursor.GoToParent()

		// No node in hierarchy
		if !moved {
			return nil
		}
	}

	return node
}
