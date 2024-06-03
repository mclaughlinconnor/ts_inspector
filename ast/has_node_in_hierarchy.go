package ast

import (
	sitter "github.com/smacker/go-tree-sitter"
)

func HasNodeInHierarchy(root *sitter.Node, nodeType string, startByte uint32, endByte uint32) *sitter.Node {
	cursor := sitter.NewTreeCursor(root)
	node := cursor.CurrentNode()
	moved := false

	for true {
		if node.StartByte() <= startByte && node.EndByte() >= endByte { // if before startByte, keep going. If after endByte, stop (backtrack?)
			moved = cursor.GoToFirstChild()
			node = cursor.CurrentNode()
		} else if node.StartByte() > startByte {
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
		if node.Type() == nodeType {
			return node
		}

		moved = cursor.GoToParent()

		// No node in hierarchy
		if !moved {
			break
		}
	}

	return nil
}
