package manipulation

import (
	"strings"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type commonNode struct {
	id             int
	kind           string
	node           *sitter.Node
	programContent *programContent
}

type childedCommonNode struct {
	commonNode
	children []nodeInterface
}

func (c *commonNode) applyAction(apply func()) []utils.TextEdit {
	c.getProgramContent().beginEditSession()
	defer c.getProgramContent().endEditSession()

	apply()

	edit := c.getProgramContent().editSession.buildLspTextEdit()

	return []utils.TextEdit{edit}
}

func (c *commonNode) editText(newText string) {
	node := c.getTsNode()

	startIndex := node.StartByte()

	oldEndIndex := node.EndByte()
	newEndIndex := startIndex + uint(len(newText))

	startPosition := node.StartPosition()

	oldEndPosition := node.EndPosition()
	newEndPosition := sitter.Point{Column: oldEndPosition.Column, Row: oldEndPosition.Row}

	lastNewLineOffset := 0
	endColumn := startPosition.Column + uint(len(newText))

	newLineCount := strings.Count(newText, "\n")
	if newLineCount > 0 {
		lastNewLineOffset = strings.LastIndex(newText, "\n")
		endColumn = uint(len(newText) - lastNewLineOffset)
	}

	newEndPosition.Column = uint(endColumn)
	newEndPosition.Row = startPosition.Row + uint(newLineCount)

	programContent := c.getProgramContent()
	programContent.editText(node.StartByte(), node.EndByte(), newText)

	ei := sitter.InputEdit{
		StartByte:      startIndex,
		OldEndByte:     oldEndIndex,
		NewEndByte:     newEndIndex,
		StartPosition:  startPosition,
		OldEndPosition: oldEndPosition,
		NewEndPosition: newEndPosition,
	}

	programContent.editTree(&ei)
}

func (c *commonNode) getActions() []action {
	return []action{}
}

func (c *commonNode) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return c.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (c *commonNode) getAstNodeOfKindAtOffset(offset uint, kind string, _ bool) (nodeInterface, bool) {
	if kind != NULL_KIND && c.getKind() != kind {
		return nil, false
	}

	if c.isUnderCursor(offset) {
		return c, true
	}

	return nil, false
}

func (c *commonNode) getKind() string {
	return c.kind
}

func (c *commonNode) getId() int {
	return c.id
}

func (c *commonNode) getText() string {
	return c.getTsNode().Utf8Text(c.getProgramText())
}

func (c *commonNode) getTsNode() *sitter.Node {
	return c.node
}

func (c *commonNode) getProgramContent() *programContent {
	return c.programContent
}

func (c *commonNode) getProgramRoot() *root {
	return c.programContent.root
}

func (c *commonNode) getProgramText() []byte {
	return c.programContent.text
}

func (c *commonNode) isUnderCursor(offset uint) bool {
	tsNode := c.getTsNode()

	return tsNode.StartByte() <= offset && offset < tsNode.EndByte()
}

func (c *commonNode) visit(exec func(nodeInterface) int) int {
	return terminalVisitor(c, exec)
}

func (c *childedCommonNode) getChildren() []nodeInterface {
	return c.children
}

func (c *childedCommonNode) visit(exec func(nodeInterface) int) int {
	return childedVisitor(c, exec)
}

func (c *childedCommonNode) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	tsNode := c.getTsNode()
	if tsNode.StartByte() > offset || offset >= tsNode.EndByte() {
		return nil, false
	}

	if first && c.getKind() == kind {
		return c, true
	}

	for _, child := range c.getChildren() {
		astNode, found := child.getAstNodeOfKindAtOffset(offset, kind, first)
		if found {
			return astNode, true
		}
	}

	if kind == NULL_KIND || c.getKind() == kind {
		return c, true
	}

	return nil, false
}

func makeCommonNode(kind string, state walkState, node *sitter.Node) commonNode {
	return commonNode{id: utils.GetNextId(ID_NAMESPACE), kind: kind, node: node, programContent: state.getProgramContent()}
}

func makeCommonChildedNode(kind string, state walkState, node *sitter.Node) childedCommonNode {
	return childedCommonNode{children: []nodeInterface{}, commonNode: makeCommonNode(kind, state, node)}
}

func childedVisitor(this childedNodeInterface, exec func(nodeInterface) int) int {
	if ret := exec(this); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, child := range this.getChildren() {
		if ret := child.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func terminalVisitor(this nodeInterface, exec func(nodeInterface) int) int {
	if ret := exec(this); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return VisitContinue
}
