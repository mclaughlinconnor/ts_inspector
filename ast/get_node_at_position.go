package ast

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func GetNamedNodeAtPosition(root *sitter.Node, offset uint) *sitter.Node {
	cursor := root.Walk()
	node := cursor.Node()
	moved := false

	for {
		// reached a terminal node that is past the cursor, go back to the parent
		if node.StartByte() > offset {
			cursor.GotoParent()
			cursor.Node()
			break
		}

		if node.StartByte() > offset || offset >= node.EndByte() {
			moved = cursor.GotoNextSibling()
			node = cursor.Node()

			if moved {
				continue
			}

			cursor.GotoParent()
			break
		}

		moved = cursor.GotoFirstChild()
		node = cursor.Node()

		if !moved {
			break
		}
	}

	for {
		node = cursor.Node()
		if node.IsNamed() {
			return node
		}

		moved = cursor.GotoParent()

		// No node in hierarchy
		if !moved {
			return nil
		}
	}
}
