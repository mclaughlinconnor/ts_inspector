package manipulation

import (
	"fmt"
	"slices"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const NULL_KIND = "__null_kind"
const ID_NAMESPACE = "manipulation"

const (
	VisitContinue = iota
	VisitAbort    = iota
	VisitSkip     = iota
)

type editSession struct {
	afterDocument  string
	beforeDocument string
}

type programContent struct {
	editSession *editSession
	root        *root
	text        []byte
	tree        *sitter.Tree
}

type nodeInterface interface {
	applyAction(apply func()) []utils.TextEdit
	editText(newText string)
	getAstNodeAtOffset(offset uint) (nodeInterface, bool)
	getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool)
	getActions() []action
	getKind() string
	getId() int
	getProgramContent() *programContent
	getProgramRoot() *root
	getProgramText() []byte
	getText() string
	getTsNode() *sitter.Node
	isUnderCursor(offset uint) bool
	visit(func(nodeInterface) int) int
}

type invertableNodeInterface interface {
	nodeInterface
	invert()
}

type commonNode struct {
	id             int
	kind           string
	node           *sitter.Node
	programContent *programContent
}

type program struct {
	commonNode
	Children []nodeInterface
}

type root struct {
	commonNode
	Program nodeInterface
}

type action struct {
	Name    string
	Perform func() []utils.TextEdit
}

type binaryExpressionOperatorType = string

type binaryExpressonOperatorStruct struct {
	AND  binaryExpressionOperatorType
	EEQ  binaryExpressionOperatorType
	EQ   binaryExpressionOperatorType
	GT   binaryExpressionOperatorType
	GTE  binaryExpressionOperatorType
	LT   binaryExpressionOperatorType
	LTE  binaryExpressionOperatorType
	NEEQ binaryExpressionOperatorType
	NEQ  binaryExpressionOperatorType
	OR   binaryExpressionOperatorType
}

var binaryExpressionOperatorEnum = binaryExpressonOperatorStruct{AND: "&&", EEQ: "===", EQ: "==", GT: ">", GTE: ">=", LT: "<", LTE: "<=", NEEQ: "!==", NEQ: "!=", OR: "||"}

type binaryExpression struct {
	commonNode
	Left     nodeInterface
	Operator binaryExpressionOperator
	Right    nodeInterface
}

type binaryExpressionOperator struct {
	commonNode
}

type boolean struct {
	commonNode
}

type expressionStatement struct {
	commonNode
	Children []nodeInterface
}

type identifier struct {
	commonNode
}

type unhandled struct {
	commonNode
	Children []nodeInterface
}

func BuildAst(content string) (*root, error) {
	utils.ResetNextId(ID_NAMESPACE)
	byteContent := []byte(content)

	rootNode, tree := utils.ParseTextWithTree(byteContent, utils.TypeScript)

	funcMap := walk.NewVisitorFuncsMap[state]()
	funcMap["binary_expression"] = visitBinaryExpression
	funcMap["expression_statement"] = visitExpressionStatement
	funcMap["false"] = visitBoolean
	funcMap["identifier"] = visitIdentifier
	funcMap["program"] = visitProgram
	funcMap["true"] = visitBoolean
	funcMap[walk.DUMMY_VISITOR_KIND] = visitUnhandled

	astRoot := root{commonNode: commonNode{kind: "root", node: rootNode, programContent: &programContent{text: byteContent, tree: tree}}}
	var ast state = &astRoot
	astRoot.getProgramContent().root = ast.(*root)

	program, err := walk.WalkTypeScript(rootNode, ast, funcMap)
	if err != nil {
		return nil, err
	}

	astRoot.Program = program

	return &astRoot, nil
}

func (b *binaryExpression) deMorgans() {
	b.invert()
	b.editText("!(" + b.getText() + ")")
}

func (b *binaryExpression) getActions() []action {
	return []action{
		{Name: "deMorgans", Perform: func() []utils.TextEdit { return b.applyAction(b.deMorgans) }},
	}
}

func (b *binaryExpression) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return b.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (b *binaryExpression) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	node := b.getTsNode()
	if node.StartByte() > offset || offset >= node.EndByte() {
		return nil, false
	}

	if first && b.getKind() == kind {
		return b, true
	}

	if b.Left != nil {
		leftNode := b.Left.getTsNode()
		if leftNode.StartByte() <= offset && offset < leftNode.EndByte() {
			return b.Left.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	operatorNode := b.Operator.getTsNode()
	if operatorNode.StartByte() <= offset && offset < operatorNode.EndByte() {
		return b.Operator.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if b.Right != nil {
		rightNode := b.Right.getTsNode()
		if rightNode.StartByte() <= offset && offset < rightNode.EndByte() {
			return b.Right.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	if kind == NULL_KIND || b.getKind() == kind {
		return b, true
	}

	return nil, false
}

func (b *binaryExpression) invert() {
	left, ok := b.Left.(invertableNodeInterface)
	if ok {
		left.invert()
	}

	b.Operator.invert()

	right, ok := b.Right.(invertableNodeInterface)
	if ok {
		right.invert()
	}
}

func (b *binaryExpression) visit(exec func(nodeInterface) int) int {
	if ret := exec(b); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := b.Left.visit(exec); ret == VisitAbort {
		return ret
	}
	if ret := b.Operator.visit(exec); ret == VisitAbort {
		return ret
	}
	if ret := b.Right.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func (c *binaryExpressionOperator) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return c.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (c *binaryExpressionOperator) getAstNodeOfKindAtOffset(offset uint, kind string, _ bool) (nodeInterface, bool) {
	if kind != NULL_KIND && c.getKind() != kind {
		return nil, false
	}

	node := c.getTsNode()
	if node.StartByte() <= offset && offset < node.EndByte() {
		return c, true
	}

	return nil, false
}

func (b *binaryExpressionOperator) getOperator() binaryExpressionOperatorType {
	return b.getText()
}

func (b *binaryExpressionOperator) invert() {
	switch b.getOperator() {
	case binaryExpressionOperatorEnum.AND:
		b.editText("||")
	case binaryExpressionOperatorEnum.EEQ:
		b.editText("!==")
	case binaryExpressionOperatorEnum.EQ:
		b.editText("!=")
	case binaryExpressionOperatorEnum.GT:
		b.editText("<=")
	case binaryExpressionOperatorEnum.GTE:
		b.editText("<")
	case binaryExpressionOperatorEnum.LT:
		b.editText(">=")
	case binaryExpressionOperatorEnum.LTE:
		b.editText(">=")
	case binaryExpressionOperatorEnum.NEEQ:
		b.editText("===")
	case binaryExpressionOperatorEnum.NEQ:
		b.editText("==")
	case binaryExpressionOperatorEnum.OR:
		b.editText("&&")
	}
}

func (b *binaryExpressionOperator) visit(exec func(nodeInterface) int) int {
	if ret := exec(b); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return VisitContinue
}

func (b *boolean) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return b.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (b *boolean) getAstNodeOfKindAtOffset(offset uint, kind string, _ bool) (nodeInterface, bool) {
	if kind != NULL_KIND && b.getKind() != kind {
		return nil, false
	}

	node := b.getTsNode()
	if node.StartByte() <= offset && offset < node.EndByte() {
		return b, true
	}

	return nil, false
}

func (b *boolean) getValue() bool {
	node := b.getTsNode()

	return node.Kind() == "true"
}

func (b *boolean) invert() {
	if b.getValue() {
		b.editText("false")
	} else {
		b.editText("true")
	}
}

func (b *boolean) visit(exec func(nodeInterface) int) int {
	if ret := exec(b); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return VisitContinue
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

func (e *editSession) buildLspTextEdit() utils.TextEdit {
	editStart := 0

	afterDocument := e.afterDocument
	beforeDocument := e.beforeDocument

	lenAfterDocument := len(afterDocument)
	lenBeforeDocument := len(beforeDocument)

	for i := range afterDocument {
		if i >= lenAfterDocument || i >= lenBeforeDocument {
			break
		}

		if afterDocument[i] == beforeDocument[i] {
			continue
		}

		editStart = i
		break
	}

	afterEditEnd := lenAfterDocument - 1
	beforeEditEnd := lenBeforeDocument - 1

	for afterEditEnd >= 0 && beforeEditEnd >= 0 {
		if afterDocument[afterEditEnd] == beforeDocument[beforeEditEnd] {
			afterEditEnd--
			beforeEditEnd--
			continue
		}

		afterEditEnd = min(afterEditEnd+1, lenAfterDocument-1)
		beforeEditEnd = min(beforeEditEnd+1, lenBeforeDocument-1)

		break
	}

	startPosition := utils.GetPositionForOffset2(beforeDocument, editStart)
	endPosition := utils.GetPositionForOffset2(beforeDocument, beforeEditEnd)

	r := utils.Range{Start: startPosition, End: endPosition}
	newText := afterDocument[editStart:afterEditEnd]

	return utils.TextEdit{Range: r, NewText: newText}
}

func (e *expressionStatement) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return e.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (e *expressionStatement) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	tsNode := e.getTsNode()
	if tsNode.StartByte() > offset || offset >= tsNode.EndByte() {
		return nil, false
	}

	if first && e.getKind() == kind {
		return e, true
	}

	for _, child := range e.Children {
		astNode, found := child.getAstNodeOfKindAtOffset(offset, kind, first)
		if found {
			return astNode, true
		}
	}

	if kind == NULL_KIND || e.getKind() == kind {
		return e, true
	}

	return nil, false
}

func (e *expressionStatement) visit(exec func(nodeInterface) int) int {
	if ret := exec(e); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, child := range e.Children {
		if ret := child.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func (i *identifier) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return i.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (i *identifier) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if kind != NULL_KIND && i.getKind() != kind {
		return nil, false
	}

	node := i.getTsNode()
	if node.StartByte() <= offset && offset < node.EndByte() {
		return i, true
	}

	return nil, false
}

func (i *identifier) visit(exec func(nodeInterface) int) int {
	if ret := exec(i); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return VisitContinue
}

func (p *programContent) beginEditSession() {
	p.editSession = &editSession{beforeDocument: string(p.text)}
}

func (p *programContent) editText(startOffset uint, endOffset uint, replacementText string) {
	p.text = slices.Replace(p.text, int(startOffset), int(endOffset), []byte(replacementText)...)
	p.updateEditSessionSnapshot(string(p.text))
}

func (p *programContent) editTree(ei *sitter.InputEdit) {
	p.tree.Edit(ei)
	p.root.visit(func(ni nodeInterface) int {
		ni.getTsNode().Edit(ei)
		return VisitContinue
	})
}

func (p *programContent) endEditSession() {
	p.editSession = nil
}

func (p *programContent) updateEditSessionSnapshot(document string) {
	p.editSession.afterDocument = document
}

func (p *program) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return p.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (p *program) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	tsNode := p.getTsNode()
	if tsNode.StartByte() > offset || offset >= tsNode.EndByte() {
		return nil, false
	}

	if first && p.getKind() == kind {
		return p, true
	}

	for _, child := range p.Children {
		astNode, found := child.getAstNodeOfKindAtOffset(offset, kind, first)
		if found {
			return astNode, true
		}
	}

	if kind == NULL_KIND || p.getKind() == kind {
		return p, true
	}

	return nil, false
}

func (p *program) visit(exec func(nodeInterface) int) int {
	if ret := exec(p); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, child := range p.Children {
		if ret := child.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func (r *root) GetAllActions(offset uint) []action {
	actions := []action{}
	r.visit(func(ni nodeInterface) int {
		if !ni.isUnderCursor(offset) {
			return VisitSkip
		}

		as := ni.getActions()
		if len(as) != 0 {
			actions = append(actions, as...)
		}

		return VisitContinue
	})

	return actions
}

func (r *root) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return r.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (r *root) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if kind == NULL_KIND || r.getKind() == kind {
		return r, true
	}

	return r.Program.getAstNodeOfKindAtOffset(offset, kind, first)
}

func (r *root) visit(exec func(nodeInterface) int) int {
	if ret := exec(r); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return r.Program.visit(exec)
}

func (u *unhandled) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return u.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (u *unhandled) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	tsNode := u.getTsNode()
	if tsNode.StartByte() > offset || offset >= tsNode.EndByte() {
		return nil, false
	}

	if first && u.getKind() == kind {
		return u, true
	}

	for _, child := range u.Children {
		astNode, found := child.getAstNodeOfKindAtOffset(offset, kind, first)
		if found {
			return astNode, true
		}
	}

	if kind == NULL_KIND || u.getKind() == kind {
		return u, true
	}

	return nil, false
}

func (u *unhandled) visit(exec func(nodeInterface) int) int {
	if ret := exec(u); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, child := range u.Children {
		if ret := child.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

type state = nodeInterface

func makeCommonNode(kind string, state state, node *sitter.Node) commonNode {
	return commonNode{id: utils.GetNextId(ID_NAMESPACE), kind: kind, node: node, programContent: state.getProgramContent()}
}

func visitBinaryExpression(node *sitter.Node, state state, indexInParent uint, funcMap walk.VisitorFuncMap[state]) (state, error) {
	binaryExpression := binaryExpression{commonNode: makeCommonNode("binaryExpression", state, node)}

	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return nil, fmt.Errorf("invalid ast: missing left")
	}

	operatorNode := node.ChildByFieldName("operator")
	if operatorNode == nil {
		return nil, fmt.Errorf("invalid ast: missing operator")
	}

	rightNode := node.ChildByFieldName("right")
	if rightNode == nil {
		return nil, fmt.Errorf("invalid ast: missing right")
	}

	left, err := walk.VisitNode(leftNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	operator := binaryExpressionOperator{commonNode: makeCommonNode("operator", state, operatorNode)}

	right, err := walk.VisitNode(rightNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	if left.getId() != state.getId() {
		binaryExpression.Left = left
	}

	binaryExpression.Operator = operator

	if right.getId() != state.getId() {
		binaryExpression.Right = right
	}

	return &binaryExpression, nil
}

func visitBoolean(node *sitter.Node, state state, indexInParent uint, funcMap walk.VisitorFuncMap[state]) (state, error) {
	boolean := boolean{commonNode: makeCommonNode("boolean", state, node)}

	return &boolean, nil
}

func visitExpressionStatement(node *sitter.Node, state state, indexInParent uint, funcMap walk.VisitorFuncMap[state]) (state, error) {
	root := expressionStatement{commonNode: makeCommonNode("expressionStatement", state, node)}

	children := []nodeInterface{}

	for i := range node.NamedChildCount() {
		child, err := walk.VisitNode(node.NamedChild(i), state, i, funcMap, false)
		if err != nil {
			return nil, err
		}

		// VisitNode returns its state if nothing is visited. Don't create cycles in the tree
		if child.getId() == state.getId() {
			continue
		}

		children = append(children, child)
	}

	root.Children = children

	return &root, nil
}

func visitIdentifier(node *sitter.Node, state state, indexInParent uint, funcMap walk.VisitorFuncMap[state]) (state, error) {
	identifier := identifier{commonNode: makeCommonNode("identifier", state, node)}

	return &identifier, nil
}

func visitProgram(node *sitter.Node, state state, indexInParent uint, funcMap walk.VisitorFuncMap[state]) (state, error) {
	root := program{commonNode: makeCommonNode("program", state, node)}

	children := []nodeInterface{}

	for i := range node.NamedChildCount() {
		child, err := walk.VisitNode(node.NamedChild(i), state, i, funcMap, false)
		if err != nil {
			return nil, err
		}

		// VisitNode returns its state if nothing is visited. Don't create cycles in the tree
		if child.getId() == state.getId() {
			continue
		}

		children = append(children, child)
	}

	root.Children = children

	return &root, nil
}

func visitUnhandled(node *sitter.Node, state state, indexInParent uint, funcMap walk.VisitorFuncMap[state]) (state, error) {
	unhandled := unhandled{commonNode: makeCommonNode("unhandled", state, node)}

	children := []nodeInterface{}

	for i := range node.NamedChildCount() {
		child, err := walk.VisitNode(node.NamedChild(i), state, i, funcMap, false)
		if err != nil {
			return nil, err
		}

		if child.getId() == state.getId() {
			continue
		}

		children = append(children, child)
	}

	unhandled.Children = children

	return &unhandled, nil
}
