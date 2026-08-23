package ast

import (
	sitter "github.com/smacker/go-tree-sitter"
)

func GetNamedNodeAtPosition(root *sitter.Node, offset uint32) *sitter.Node {
	cursor := sitter.NewTreeCursor(root)
	node := cursor.CurrentNode()
	moved := false

	for {
		// reached a terminal node that is past the cursor, go back to the parent
		if node.StartByte() > offset {
			cursor.GoToParent()
			cursor.CurrentNode()
			break
		}

		if node.StartByte() > offset || offset >= node.EndByte() {
			moved = cursor.GoToNextSibling()
			node = cursor.CurrentNode()

			if moved {
				continue
			}

			cursor.GoToParent()
			break
		}

		moved = cursor.GoToFirstChild()
		node = cursor.CurrentNode()

		if !moved {
			break
		}
	}

	for {
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
}
