package manipulation

import (
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type childedCommonNode struct {
	commonNode
	children []nodeInterface
}

type childedNodeInterface interface {
	nodeInterface
	getChildren() []nodeInterface
}

type commonNode struct {
	element        *element
	id             int
	kind           string
	programContent *programContent
	stagedElement  *element
}

type invertableNodeInterface interface {
	nodeInterface
	invert()
}

type nodeInterface interface {
	applyAction(apply func()) []utils.TextEdit
	// beginEditSession()
	// commitEditSession()
	// dropEditSession()
	editText(newText string)
	getAstNodeAtOffset(offset uint) (nodeInterface, bool)
	getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool)
	getActions() []action
	getElement() *element
	getKind() string
	getId() int
	getProgramContent() *programContent
	getProgramRoot() *root
	getProgramText() []byte
	getText() string
	isUnderCursor(offset uint) bool
	visit(func(nodeInterface) int) int
}

func (c *commonNode) applyAction(apply func()) []utils.TextEdit {
	c.getProgramContent().beginEditSession()
	defer c.getProgramContent().endEditSession()

	apply()

	edit := c.getProgramContent().editSession.buildLspTextEdit()

	return []utils.TextEdit{edit}
}

func (c *commonNode) editText(newText string) {
	node := c.getElement()

	startIndex := node.getStartOffset()

	oldEndIndex := node.getEndOffset()
	newEndIndex := startIndex + uint(len(newText))

	programContent := c.getProgramContent()
	programContent.editText(node.getStartOffset(), node.getEndOffset(), newText)

	edit := elementEdit{
		startOffset:  startIndex,
		oldEndOffset: oldEndIndex,
		newEndOffset: newEndIndex,
	}

	programContent.editTree(&edit)
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

func (c *commonNode) getElement() *element {
	return c.element
}

func (c *commonNode) getKind() string {
	return c.kind
}

func (c *commonNode) getId() int {
	return c.id
}

func (c *commonNode) getText() string {
	element := c.getElement()
	return string(c.getProgramText()[element.startOffset:element.endOffset])
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
	tsNode := c.getElement()

	return tsNode.getStartOffset() <= offset && offset < tsNode.getEndOffset()
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
	tsNode := c.getElement()
	if tsNode.getStartOffset() > offset || offset >= tsNode.getEndOffset() {
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
	return commonNode{id: utils.GetNextId(ID_NAMESPACE), kind: kind, element: elementFromNode(node), programContent: state.getProgramContent()}
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
