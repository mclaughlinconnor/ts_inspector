package ast

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func HasNodeInHierarchy(root *sitter.Node, nodeType string, startByte uint, endByte uint) *sitter.Node {
	cursor := root.Walk()
	node := cursor.Node()
	moved := false

	for {
		if node.StartByte() <= startByte && node.EndByte() > endByte { // if before startByte, keep going. If after endByte, stop (backtrack?)
			moved = cursor.GotoFirstChild()
			node = cursor.Node()
		} else if node.StartByte() > startByte {
			cursor.GotoParent() // reached a terminal node that is past the cursor, go back to the parent
			break
		} else {
			moved = cursor.GotoNextSibling()
			node = cursor.Node()
		}

		if !moved {
			break
		}
	}

	for {
		node = cursor.Node()
		if node.Kind() == nodeType {
			return node
		}

		moved = cursor.GotoParent()

		// No node in hierarchy
		if !moved {
			break
		}
	}

	return nil
}
