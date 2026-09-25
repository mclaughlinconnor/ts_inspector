package manipulation

import (
	"fmt"
	"ts_inspector/interfaces"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type AstManipulationNode = nodeInterface

type childedCommonNode struct {
	commonNode
	_self    childedNodeInterface
	children []nodeInterface
}

type childedNodeInterface interface {
	nodeInterface
	impl[childedNodeInterface]
	getChildren() []nodeInterface
	visitChildren(func(nodeInterface) int) int
}

type commonNode struct {
	_self          nodeInterface
	cfg            *Cfg
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
	GetEndOffset() uint
	GetRange() utils.Range
	GetStartOffset() uint

	applyAction(apply func()) ([]utils.TextEdit, error)
	beginEditSession() error
	buildCfgBlock() (bool, error)
	commitEditSession() error
	dropEditSession() error
	editText(newText string)
	getActions() []action
	getAnalysis() []interfaces.Analysis
	getAstNodeAtOffset(offset uint) (nodeInterface, bool)
	getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool)
	getCfg() *Cfg
	getEndOffset() uint
	getElement() *element
	getId() int
	getKind() string
	getNode() impl[nodeInterface]
	getProgramContent() *programContent
	getProgramRoot() *Ast
	getProgramText() string
	getRange() utils.Range
	getStagedElement() *element
	getStartOffset() uint
	getText() string
	hasConstantExpression() bool
	hasConstantFalse() bool
	hasConstantTrue() bool
	hasEditSession() bool
	isUnderCursor(offset uint) bool
	setElement(element *element)
	setStagedElement(element *element)
	visit(func(nodeInterface) int) int

	_buildCfgBlock() (bool, error)
}

func (c *commonNode) GetEndOffset() uint {
	return c.getEndOffset()
}

func (c *commonNode) GetRange() utils.Range {
	return c.getRange()
}

func (c *commonNode) GetStartOffset() uint {
	return c.getStartOffset()
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

func (c *commonNode) buildCfgBlock() (bool, error) {
	return c.getImpl()._buildCfgBlock()
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
	startIndex := c.getStartOffset()

	oldEndIndex := c.getEndOffset()
	newEndIndex := startIndex + uint(len(newText))

	programContent := c.getImpl().getProgramContent()
	programContent.editText(c.getStartOffset(), c.getEndOffset(), newText)

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
	return []interfaces.Analysis{
		interfaces.NewAnalysis("debug", c.getRange(), interfaces.AnalysisSeverity.Warning, c.getKind(), nil),
	}
	// return []interfaces.Analysis{}
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

func (c *commonNode) getCfg() *Cfg {
	return c.cfg
}

func (c *commonNode) getElement() *element {
	stagedElement := c.getImpl().getStagedElement()
	if stagedElement != nil {
		return stagedElement
	}

	return c.element
}

func (c *commonNode) getEndOffset() uint {
	return c.getImpl().getElement().getEndOffset()
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

	start := utils.GetPositionForOffset(content, c.getImpl().getStartOffset())
	end := utils.GetPositionForOffset(content, c.getImpl().getEndOffset())

	return utils.Range{End: end, Start: start}
}

func (c *commonNode) getStartOffset() uint {
	return c.getImpl().getElement().getStartOffset()
}

func (c *commonNode) getText() string {
	element := c.getImpl().getElement()
	return c.getImpl().getProgramText()[element.startOffset:element.endOffset]
}

func (c *commonNode) hasConstantExpression() bool {
	return false
}

func (c *commonNode) hasConstantFalse() bool {
	return false
}

func (c *commonNode) hasConstantTrue() bool {
	return false
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

func (c *commonNode) isUnderCursor(offset uint) bool {
	return c.getStartOffset() <= offset && offset < c.getEndOffset()
}

func (c *commonNode) setElement(element *element) {
	if c.getImpl().getStagedElement() != nil {
		c.getImpl().setStagedElement(element)
	}

	c.element = element
}

func (c *commonNode) setImpl(impl nodeInterface) {
	c._self = impl
}

func (c *commonNode) setStagedElement(element *element) {
	c.stagedElement = element
}

func (c *commonNode) visit(exec func(nodeInterface) int) int {
	return terminalVisitor(c, exec)
}

func (c *commonNode) _buildCfgBlock() (bool, error) {
	return false, nil
}

func (c *childedCommonNode) _buildCfgBlock() (bool, error) {
	built := false
	for _, child := range c.getChildren() {
		b, err := child.buildCfgBlock()
		built = built || b

		if err != nil {
			return built, err
		}
	}

	return built, nil
}

func (c *childedCommonNode) getChildren() []nodeInterface {
	return c.children
}

func (c *childedCommonNode) getImpl() childedNodeInterface {
	return c._self
}

func (c *childedCommonNode) setImpl(impl childedNodeInterface) {
	c._self = impl
	c.commonNode.setImpl(impl)
}

func (c *childedCommonNode) visit(exec func(nodeInterface) int) int {
	return childedVisitor(c, exec)
}

func (c *childedCommonNode) visitChildren(exec func(nodeInterface) int) int {
	for _, child := range c.getImpl().getChildren() {
		if ret := child.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
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
	return commonNode{
		id:             utils.GetNextId(ID_NAMESPACE),
		kind:           kind,
		element:        elementFromNode(node),
		programContent: state.getProgramContent(),
		cfg:            state.getCfg(),
	}
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

	if ret := this.visitChildren(exec); ret == VisitAbort {
		return ret
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

func buildFunctionCfg(this nodeInterface, name string, body nodeInterface) error {
	blockName := "Function " + name

	cfg := this.getCfg()
	cfg.addInstruction(instructionAssign, name, this, "")

	currentCfg := &FunctionCfg{blocks: []*CfgBlock{}, Node: this, Kind: "function"}
	cfg.allCfg = append(cfg.allCfg, currentCfg)
	cfg.cfgStack.Push(currentCfg)

	start := cfg.currentCfg().addBlock(blockName + " start")
	end := cfg.currentCfg().addBlock(blockName + " end")

	cfg.currentCfg().Start = start
	cfg.currentCfg().End = end

	prevCurrent := cfg.current
	cfg.current = start

	_, err := body.buildCfgBlock()
	if err != nil {
		return err
	}

	cfg.currentCfg().addEdge(cfg.current, end)

	cfg.cfgStack.Pop()
	cfg.current = prevCurrent

	return nil
}

func validateChildNodeExists(node *sitter.Node, child string) (*sitter.Node, error) {
	return validateNodeExists(node.ChildByFieldName(child), child)
}

func validateNodeExists(node *sitter.Node, name string) (*sitter.Node, error) {
	if node == nil {
		return nil, fmt.Errorf("invalid ast: missing %s", name)
	}

	return node, nil
}

func isNode[T nodeInterface](node nodeInterface) (T, bool) {
	if node == nil || node.getNode() == nil {
		var zero T
		return zero, false
	}

	n, yes := node.getNode().getImpl().(T)
	return n, yes
}
