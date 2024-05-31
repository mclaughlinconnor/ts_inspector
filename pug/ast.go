package pug

import (
	sitter "github.com/smacker/go-tree-sitter"
)

func visitAttributeName(node *sitter.Node, state *State) {
	r := getRange(node)
	pushRange(state, node.Content(content), &ATTRIBUTE_NAME, &r)
}

func visitAttributes(node *sitter.Node, state *State) {
	cursor := sitter.NewTreeCursor(node)
	cursor.GoToFirstChild()
	for { // TODO: what is attribute.type() here
		attribute := cursor.CurrentNode()
		if !attribute.IsNamed() {
			if cursor.GoToNextSibling() {
				continue
			} else {
				break
			}
		}

		index := 0

		attributeName := attribute.Child(index)
		if attributeName == nil {
			if cursor.GoToNextSibling() {
				continue
			} else {
				break
			}
		}

		visitAttributeName(attributeName, state)

		index = index + 1
		attributeValue := attribute.NamedChild(index)

		cursor.GoToNextSibling()
		nextSibling := cursor.CurrentNode()

		if attributeValue != nil {
			r := offsetPreviousRange(state, 1)
			pushRange(state, "=", &EQUALS, &r)
			traverseTree(attributeValue, state)
		} else if nextSibling.Content(content) == "=" {
			// if "attr=" has been typed, we still want the = in case that's where the cursor is
			r := offsetPreviousRange(state, 1)
			pushRange(state, "=", &EQUALS, &r)
		}

		lastRange := state.Ranges[len(state.Ranges)-1]
		spaceEnd := lastRange.PugEnd + 1
		for {
			node := cursor.CurrentNode()

			if node.Content(content) == "," || node.Type() == "ERROR" {
				pushRange(state, " ", &SPACE, &NodeRange{StartIndex: spaceEnd + 1, EndIndex: node.StartByte()})
			} else if node.Content(content) == ")" || node.IsNamed() {
				if node.IsNamed() {
					spaceEnd = node.StartByte() - 1
				} else {
					spaceEnd = node.StartByte()
				}
				break
			}
			if node.Type() == "attribute" || !cursor.GoToNextSibling() {
				break
			}
		}

		pushRange(state, " ", &SPACE, &NodeRange{StartIndex: (state.Ranges[len(state.Ranges)-1].PugEnd) + 1, EndIndex: spaceEnd})
	}
}

func visitTagName(node *sitter.Node, state *State) {
	pugRange := getRange(node)
	htmlLen := len(state.HtmlText)
	toPush := node.Content(content)

	r := Range{
		HtmlStart: uint32(htmlLen),
		HtmlEnd:   uint32(htmlLen + len(toPush)),
		NodeType:  TAG_NAME,
		PugStart:  pugRange.StartIndex,
		PugEnd:    pugRange.EndIndex,
	}

	state.Ranges = append(state.Ranges, r)
	state.HtmlText += toPush
}

/**
 * @param {Parser.SyntaxNode[]} nodes
 * @param {State} state
 * @returns {void}
 */
func visitIdClass(nodes []*sitter.Node, state *State) {
	start := true

	for _, node := range nodes {
		if !start {
			pushRange(state, " ", nil, nil)
		}

		r := getRange(node)
		r.StartIndex = r.StartIndex + 1
		text := node.Content(content)[1:]

		pushRange(state, text, &ID_CLASS, &r)

		start = false
	}
}

func handleClosingTagName(state *State, name string) {
	var offset int32 = 0
	if IsVoidElement(name) {
		pushRange(state, "/", nil, nil)
		r := offsetPreviousRange(state, -1)
		pushRange(state, "", &EMPTY, &r)
		offset = offset - 1
	}
	pushRange(state, ">", nil, nil)
	r := offsetPreviousRange(state, offset)
	pushRange(state, "", &EMPTY, &r)
}

/**
 * @param {Parser.SyntaxNode} node
 * @param {State} state
 * @returns {void}
 */
func visitTag(node *sitter.Node, state *State) {
	cursor := sitter.NewTreeCursor(node)
	cursor.GoToFirstChild()

	var name string

	handledClosingTagName := false

	classes := []*sitter.Node{}
	ids := []*sitter.Node{}

	if node.Child(0).Type() == "tag_name" {
		startRange := getRange(node.Child(0))
		startRange.EndIndex = startRange.StartIndex
		startRange.EndPosition = startRange.StartPosition
		pushRange(state, "", &EMPTY, &startRange)
		pushRange(state, "<", nil, nil)

		traverseTree(node.Child(0), state)
		name = node.Child(0).Content(content)
	} else {
		pushRange(state, "<", nil, nil)
		pushRange(state, "div", nil, nil)
		name = "div"
	}

	for {
		childNode := cursor.CurrentNode()
		if childNode.Type() == "tag_name" {
			if cursor.GoToNextSibling() {
				continue
			} else {
				break
			}
		}

		if childNode.Type() == "class" {
			classes = append(classes, childNode)
			if cursor.GoToNextSibling() {
				continue
			} else {
				break
			}
		}

		if childNode.Type() == "id" {
			ids = append(ids, childNode)
			if cursor.GoToNextSibling() {
				continue
			} else {
				break
			}
		}

		if childNode.Type() == "attributes" {
			if len(classes) != 0 {
				pushRange(state, " class='", nil, nil)
				visitIdClass(classes, state)
				pushRange(state, "'", nil, nil)
			}

			if len(ids) != 0 {
				pushRange(state, " id='", nil, nil)
				visitIdClass(ids, state)
				pushRange(state, "'", nil, nil)
			}

			r := offsetPreviousRange(state, 0)
			pushRange(state, " ", &SPACE, &r)
			traverseTree(childNode, state)
			// if (!childNode.isNamed) {
			s := getRange(childNode.Child(int(childNode.ChildCount()) - 1))
			pushRange(state, " ", &SPACE, &s)
			// }

			if cursor.GoToNextSibling() {
				continue
			} else {
				break
			}
		}

		if !handledClosingTagName {
			handleClosingTagName(state, name)
			handledClosingTagName = true
		}

		// found something else that needs no extra handling
		traverseTree(childNode, state)

		if !cursor.GoToNextSibling() {
			break
		}
	}

	if !handledClosingTagName {
		handleClosingTagName(state, name)
	}

	if !IsVoidElement(name) {
		pushRange(state, "</", nil, nil)
		pushRange(state, name, nil, nil)
		pushRange(state, ">", nil, nil)
	}
}

/**
 * @param {Parser.SyntaxNode} node
 * @param {State} state
 * @returns {void}
 */
func visitConditional(node *sitter.Node, state *State) {
	conditionalCursor := sitter.NewTreeCursor(node)
	conditionalCursor.GoToFirstChild()

	conditionalCursor.GoToFirstChild()
	conditionalCursor.GoToNextSibling()

	if conditionalCursor.CurrentNode().Type() == "javascript" {
		condition := conditionalCursor.CurrentNode()

		pushRange(state, "<script>return ", nil, nil)
		r := getRange(condition)
		pushRange(state, condition.Content(content), &JAVASCRIPT, &r)
		pushRange(state, ";</script>", nil, nil)
		conditionalCursor.GoToNextSibling()
	}

	conditionalCursor.GoToNextSibling()

	conditionalCursor.GoToFirstChild()
	for conditionalCursor.GoToNextSibling() {
		traverseTree(conditionalCursor.CurrentNode(), state)
	}

}

/**
 * @param {Parser.SyntaxNode} node
 * @param {State} state
 * @returns {void}
 */
func visitPipe(node *sitter.Node, state *State) {
	cursor := sitter.NewTreeCursor(node)
	cursor.GoToFirstChild()

	for cursor.GoToNextSibling() {
		if cursor.CurrentNode().IsNamed() {
			traverseTree(cursor.CurrentNode(), state)
		}
	}
}

func visitTagInterpolation(node *sitter.Node, state *State) {
	interpolationCursor := sitter.NewTreeCursor(node)

	interpolationCursor.GoToFirstChild()
	interpolationCursor.GoToNextSibling()
	interpolationCursor.GoToFirstChild()

	for {
		traverseTree(interpolationCursor.CurrentNode(), state)
		if !interpolationCursor.GoToNextSibling() {
			break
		}
	}
}

func visitFilename(node *sitter.Node, state *State) {
	pushRange(state, "<a href='", nil, nil)
	r := getRange(node)
	pushRange(state, node.Content(content), &FILENAME, &r)
	pushRange(state, "'></a>", nil, nil)
}

func visitExtendsInclude(node *sitter.Node, state *State) {
	cursor := sitter.NewTreeCursor(node)
	cursor.GoToFirstChild()
	for {
		traverseTree(cursor.CurrentNode(), state)
		if !cursor.GoToNextSibling() {
			break
		}
	}
}

/**
 * @param {Parser.SyntaxNode} node
 * @param {State} state
 * @returns {void}
 */
func visitCaseWhen(node *sitter.Node, state *State) {
	cursor := sitter.NewTreeCursor(node)
	cursor.GoToFirstChild()

	for {
		child := cursor.CurrentNode()
		if child.Type() == "javascript" {
			pushRange(state, "<script>return ", nil, nil)
			r := getRange(child)
			pushRange(state, child.Content(content), &JAVASCRIPT, &r)
			pushRange(state, ";</script>", nil, nil)
		} else {
			traverseTree(child, state)
		}
		if !cursor.GoToNextSibling() {
			break
		}
	}
}

/**
 * @param {Parser.SyntaxNode} node
 * @param {State} state
 * @returns {void}
 */
func visitUnbufferedCode(node *sitter.Node, state *State) {
	cursor := sitter.NewTreeCursor(node)
	cursor.GoToFirstChild()

	for {
		child := cursor.CurrentNode()
		if child.Type() == "javascript" {
			pushRange(state, "<script>", nil, nil)
			r := getRange(child)
			pushRange(state, child.Content(content), &JAVASCRIPT, &r)
			pushRange(state, ";</script>", nil, nil)
		} else {
			traverseTree(child, state)
		}

		if !cursor.GoToNextSibling() {
			break
		}
	}
}

/**
 * @param {Parser.SyntaxNode} node
 * @param {State} state
 * @returns {void}
 */
func visitMixinDefinition(node *sitter.Node, state *State) {
	if node.NamedChildCount() == 2 {
		return
	}

	index := 2 // skip the keyword and the name

	pushRange(state, "<ng-template ", nil, nil)

	if node.NamedChildCount() >= uint32(index-1) {
		if node.NamedChild(index).Type() == "mixin_attributes" {
			attributes := sitter.NewTreeCursor(node.Child(index))

			attributes.GoToFirstChild()
			for {
				attribute := attributes.CurrentNode()
				if !attribute.IsNamed() { // things like the comma in between
					if attributes.GoToNextSibling() {
						continue
					} else {
						break
					}
				}

				pushRange(state, "let-", nil, nil)
				r := getRange(attribute)
				pushRange(state, attribute.Content(content), &ATTRIBUTE, &r)
				pushRange(state, " ", nil, nil)

				if !attributes.GoToNextSibling() {
					break
				}
			}
			index++
		}
	}

	pushRange(state, ">", nil, nil)

	// Should just be the mixin content now
	traverseTree(node.NamedChild(index), state)

	pushRange(state, "</ng-template>", nil, nil)
}

/**
 * @param {Parser.SyntaxNode} node
 * @param {State} state
 * @returns {void}
 */
func visitBufferedCode(node *sitter.Node, state *State) {
	cursor := sitter.NewTreeCursor(node)
	cursor.GoToFirstChild()

	for {
		child := cursor.CurrentNode()
		if child.Type() == "javascript" {
			pushRange(state, "<script>return ", nil, nil)
			r := getRange(child)
			pushRange(state, child.Content(content), &JAVASCRIPT, &r)
			pushRange(state, ";</script>", nil, nil)
		} else {
			traverseTree(child, state)
		}

		if !cursor.GoToNextSibling() {
			break
		}
	}
}

/**
 * @param {Parser.SyntaxNode} node
 * @param {State} state
 * @returns {void}
 */
