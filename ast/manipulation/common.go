package manipulation

import (
	"fmt"
	"ts_inspector/interfaces"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type childedCommonNode struct {
	commonNode
	_self    childedNodeInterface
	children []nodeInterface
}

type childedNodeInterface interface {
	nodeInterface
	impl[childedNodeInterface]
	getChildren() []nodeInterface
}

type commonNode struct {
	_self          nodeInterface
	element        *element
	id             int
	kind           string
	programContent *programContent
	stagedElement  *element
}

type impl[T nodeInterface] interface {
	getImpl() T
}

type invertableNodeInterface interface {
	nodeInterface
	invert()
}

type nodeInterface interface {
	applyAction(apply func()) ([]utils.TextEdit, error)
	beginEditSession() error
	commitEditSession() error
	dropEditSession() error
	editText(newText string)
	getActions() []action
	getAnalysis() []interfaces.Analysis
	getAstNodeAtOffset(offset uint) (nodeInterface, bool)
	getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool)
	getElement() *element
	getId() int
	getKind() string
	getNode() impl[nodeInterface]
	getProgramContent() *programContent
	getProgramRoot() *Ast
	getProgramText() string
	getRange() utils.Range
	getStagedElement() *element
	getText() string
	hasEditSession() bool
	isBinaryExpression() (*binaryExpression, bool)
	isBoolean() (*boolean, bool)
	isElseClause() (*elseClause, bool)
	isExpressionStatement() (*expressionStatement, bool)
	isIdentifier() (*identifier, bool)
	isIfStatement() (*ifStatement, bool)
	isInvertable() (invertableNodeInterface, bool)
	isProgram() (*program, bool)
	isRoot() (*Ast, bool)
	isUnaryExpression() (*unaryExpression, bool)
	isUnderCursor(offset uint) bool
	isUnhandled() (*unhandled, bool)
	setElement(element *element)
	setStagedElement(element *element)
	visit(func(nodeInterface) int) int
}

func (c *commonNode) applyAction(apply func()) ([]utils.TextEdit, error) {
	err := c.getImpl().getProgramContent().beginEditSession()
	if err != nil {
		return []utils.TextEdit{}, err
	}

	apply()

	edits := c.getImpl().getProgramContent().editSession.buildLspTextEdits()

	err = c.getImpl().getProgramContent().dropEditSession()
	if err != nil {
		return []utils.TextEdit{}, err
	}

	return edits, nil
}

func (c *commonNode) beginEditSession() error {
	if c.hasEditSession() {
		return fmt.Errorf("tried to start an edit session when there is already an edit session in progress")
	}

	stagedElement := c.getImpl().getElement().copy()
	c.getImpl().setStagedElement(stagedElement)

	return nil
}

func (c *commonNode) commitEditSession() error {
	if !c.hasEditSession() {
		return fmt.Errorf("tried to commit an edit session when there is no edit session in progress")
	}

	element := c.getImpl().getElement()
	c.setElement(element)

	return nil
}

func (c *commonNode) dropEditSession() error {
	if !c.hasEditSession() {
		return fmt.Errorf("tried to drop an edit session when there is no edit session in progress")
	}

	c.getImpl().setStagedElement(nil)

	return nil
}

func (c *commonNode) editText(newText string) {
	element := c.getImpl().getElement()

	startIndex := element.getStartOffset()

	oldEndIndex := element.getEndOffset()
	newEndIndex := startIndex + uint(len(newText))

	programContent := c.getImpl().getProgramContent()
	programContent.editText(element.getStartOffset(), element.getEndOffset(), newText)

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

func (c *commonNode) getAnalysis() []interfaces.Analysis {
	return []interfaces.Analysis{}
}

func (c *commonNode) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return c.getImpl().getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (c *commonNode) getAstNodeOfKindAtOffset(offset uint, kind string, _ bool) (nodeInterface, bool) {
	if kind != NULL_KIND && c.getImpl().getKind() != kind {
		return nil, false
	}

	if c.getImpl().isUnderCursor(offset) {
		return c, true
	}

	return nil, false
}

func (c *commonNode) getElement() *element {
	stagedElement := c.getImpl().getStagedElement()
	if stagedElement != nil {
		return stagedElement
	}

	return c.element
}

func (c *commonNode) getKind() string {
	return c.kind
}

func (c *commonNode) getId() int {
	return c.id
}

func (c *commonNode) getImpl() nodeInterface {
	if c._self == nil {
		return c
	}

	return c._self
}

func (c *commonNode) getNode() impl[nodeInterface] {
	return c
}

func (c *commonNode) getRange() utils.Range {
	content := c.getImpl().getProgramText()

	start := utils.GetPositionForOffset(content, c.getImpl().getElement().getStartOffset())
	end := utils.GetPositionForOffset(content, c.getImpl().getElement().getEndOffset())

	return utils.Range{End: end, Start: start}
}

func (c *commonNode) getText() string {
	element := c.getImpl().getElement()
	return c.getImpl().getProgramText()[element.startOffset:element.endOffset]
}

func (c *commonNode) getProgramContent() *programContent {
	return c.programContent
}

func (c *commonNode) getProgramRoot() *Ast {
	return c.getImpl().getProgramContent().getRoot()
}

func (c *commonNode) getProgramText() string {
	return c.getImpl().getProgramContent().getText()
}

func (c *commonNode) getStagedElement() *element {
	return c.stagedElement
}

func (c *commonNode) hasEditSession() bool {
	stagedElement := c.getImpl().getStagedElement()
	return stagedElement != nil
}

func (c *commonNode) isBinaryExpression() (*binaryExpression, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*binaryExpression)
	return n, yes
}

func (c *commonNode) isBoolean() (*boolean, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*boolean)
	return n, yes
}

func (c *commonNode) isElseClause() (*elseClause, bool) {
	node := c.getImpl().getNode().getImpl()
	n, yes := node.(*elseClause)

	return n, yes
}

func (c *commonNode) isExpressionStatement() (*expressionStatement, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*expressionStatement)
	return n, yes
}

func (c *commonNode) isIdentifier() (*identifier, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*identifier)
	return n, yes
}

func (c *commonNode) isIfStatement() (*ifStatement, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*ifStatement)
	return n, yes
}

func (c *commonNode) isInvertable() (invertableNodeInterface, bool) {
	n, yes := c.getImpl().getNode().getImpl().(invertableNodeInterface)
	return n, yes
}

func (c *commonNode) isProgram() (*program, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*program)
	return n, yes
}

func (c *commonNode) isRoot() (*Ast, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*Ast)
	return n, yes
}

func (c *commonNode) isUnaryExpression() (*unaryExpression, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*unaryExpression)
	return n, yes
}

func (c *commonNode) isUnderCursor(offset uint) bool {
	element := c.getImpl().getElement()

	return element.getStartOffset() <= offset && offset < element.getEndOffset()
}

func (c *commonNode) isUnhandled() (*unhandled, bool) {
	n, yes := c.getImpl().getNode().getImpl().(*unhandled)
	return n, yes
}

func (c *commonNode) setElement(element *element) {
	if c.getImpl().getStagedElement() != nil {
		c.getImpl().setStagedElement(element)
	}

	c.element = element
}

func (c *commonNode) setStagedElement(element *element) {
	c.stagedElement = element
}

func (c *commonNode) visit(exec func(nodeInterface) int) int {
	return terminalVisitor(c, exec)
}

func (c *childedCommonNode) getChildren() []nodeInterface {
	return c.children
}

func (c *childedCommonNode) getImpl() childedNodeInterface {
	return c._self
}

func (c *childedCommonNode) visit(exec func(nodeInterface) int) int {
	return childedVisitor(c, exec)
}

func (c *childedCommonNode) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	tsNode := c.getImpl().getElement()
	if tsNode.getStartOffset() > offset || offset >= tsNode.getEndOffset() {
		return nil, false
	}

	if first && c.getImpl().getKind() == kind {
		return c, true
	}

	for _, child := range c.getImpl().getChildren() {
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

	for _, child := range this.getImpl().getChildren() {
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
