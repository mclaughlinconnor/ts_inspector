package cfg

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type InstructionKind = int

const (
	InstructionJump InstructionKind = iota
	InstructionBranch
	InstructionCall
	InstructionAssign
)

type Instruction struct {
	kind  InstructionKind
	left  string
	Node  *sitter.Node
	right string
	text  string
}

type Block struct {
	Node         *sitter.Node
	Instructions []*Instruction
	Before       []*Block
	After        []*Block
	Label        string
}

type FunctionCFG struct {
	Blocks []*Block
	End    *Block
	Node   *sitter.Node
	Start  *Block
	Type   string
}

type State struct {
	content       []byte
	AllCfg        []*FunctionCFG
	breakStack    utils.Stack[*Block]
	cfgStack      utils.Stack[*FunctionCFG]
	continueStack utils.Stack[*Block]
	current       *Block
}

func newState(content []byte) *State {
	continueStack := utils.NewStack[*Block]()
	cfgStack := utils.NewStack[*FunctionCFG]()

	return &State{
		AllCfg:        []*FunctionCFG{},
		continueStack: *continueStack,
		cfgStack:      *cfgStack,
		content:       content,
	}
}

func (b *Block) countDownwardEdges(seen map[*Block]bool) int {
	seen[b] = true
	count := len(b.After)

	for _, a := range b.After {
		if seen[a] {
			continue
		}

		count += a.countDownwardEdges(seen)
	}

	return count
}

func (b *Block) getDownwardNodes(blocks map[*Block]bool) {
	blocks[b] = true

	for _, a := range b.After {
		if blocks[a] {
			continue
		}

		a.getDownwardNodes(blocks)
	}
}

func (b *Block) getExpressionNode() *sitter.Node {
	node := b.Node

	if node == nil {
		return nil
	}

	if node.Type() == "parenthesized_expression" {
		node = node.NamedChild(0)
		if node == nil {
			return nil
		}
	}

	return node
}

//nolint:unused
func (b *Block) hasConstantExpression(content []byte) bool {
	node := b.getExpressionNode()
	if node == nil {
		return false
	}

	return hasConstantExpression(node, content)
}

func (b *Block) hasConstantFalse(content []byte) bool {
	node := b.getExpressionNode()
	if node == nil {
		return false
	}

	return hasConstantFalse(node, content)
}

func (b *Block) hasConstantTrue(content []byte) bool {
	node := b.getExpressionNode()
	if node == nil {
		return false
	}

	return hasConstantTrue(node, content)
}

func (b *Block) CalculateCyclomaticComplexity() int {
	return b.CountDownwardEdges() - b.CountDownwardNodes() + 2
}

func (b *Block) CountDownwardEdges() int {
	seen := map[*Block]bool{}

	return b.countDownwardEdges(seen)
}

func (b *Block) CountDownwardNodes() int {
	blocks := map[*Block]bool{}
	b.getDownwardNodes(blocks)

	return len(slices.Collect(maps.Keys(blocks)))
}

func (s *State) cfg() *FunctionCFG {
	return *s.cfgStack.Peek()
}

func (s *State) pushLoopBlocks(continueBlock *Block, breakBlock *Block) {
	s.continueStack.Push(continueBlock)
	s.breakStack.Push(breakBlock)
}

func (s *State) peekBreakBlock() *Block {
	b := s.breakStack.Peek()
	if b != nil {
		return *b
	}

	return nil
}

func (s *State) peekContinueBlock() *Block {
	b := s.continueStack.Peek()
	if b != nil {
		return *b
	}

	return nil
}

func (s *State) popLoopBlocks() {
	s.continueStack.Pop()
	s.breakStack.Pop()
}

func (cfg *FunctionCFG) AddEdge(from *Block, to *Block) {
	from.After = append(from.After, to)
	to.Before = append(to.Before, from)
}

func (cfg *FunctionCFG) AddBlock(label string) *Block {
	block := Block{Label: label}
	cfg.Blocks = append(cfg.Blocks, &block)

	return &block
}

func (cfg *FunctionCFG) CalculateCyclomaticComplexity() int {
	return cfg.Start.CalculateCyclomaticComplexity()
}

func (cfg *FunctionCFG) CountDownwardEdges() int {
	return cfg.Start.CountDownwardEdges()
}

func (cfg *FunctionCFG) CountDownwardNodes() int {
	return cfg.Start.CountDownwardNodes()
}

func (state *State) AddInstruction(kind InstructionKind, left string, node *sitter.Node, right string, content []byte) {
	if len(state.current.After) != 0 {
		current := state.cfg().AddBlock("Continuation")
		state.current = current
	}

	instruction := Instruction{kind, left, node, right, node.Content(content)}
	state.current.Instructions = append(state.current.Instructions, &instruction)
}

type visitorFunction = func(state *State, node *sitter.Node, content []byte) error

var visitMap = map[string]visitorFunction{
	"arguments":            handleNamedChildren,
	"arrow_function":       handleFunction,
	"await_expression":     handleNamedChildren,
	"break_statement":      handleBreak,
	"call_expression":      handleCall,
	"class_declaration":    handleClass,
	"continue_statement":   handleContinue,
	"else_clause":          handleNamedChildren,
	"export_statement":     handleNamedChildren,
	"expression_statement": handleNamedChildren,
	"for_in_statement":     handleForIn,
	"function_declaration": handleFunction,
	"if_statement":         handleIf,
	"lexical_declaration":  handleVariableDeclaration,
	"member_expression":    handleNamedChildren,
	"method_definition":    handleFunction,
	"new_expression":       handleNamedChildren,
	"program":              handleProgram,
	"return_statement":     handleReturn,
	"statement_block":      handleNamedChildren,
	"variable_declaration": handleVariableDeclaration,
	"while_statement":      handleWhile,
}

var funcMap = walk.NewVisitorFuncsMap[*State]()

func InitBuilder() {
	for k, v := range visitMap {
		funcMap[k] = func(node *sitter.Node, state *State, indexInParent int, funcMap walk.VisitorFuncMap[*State]) (*State, error) {
			err := v(state, node, state.content)
			if err != nil {
				return nil, err
			}

			return state, nil
		}
	}
}

func build(state *State, root *sitter.Node, _ []byte) error {
	_, err := walk.WalkTypeScriptShallow(root, state, funcMap)
	return err
}

func handleClass(state *State, node *sitter.Node, content []byte) error {
	body := node.ChildByFieldName("body")
	if body == nil {
		return nil
	}

	for i := range body.NamedChildCount() {
		child := body.NamedChild(int(i))
		if child.Type() != "method_definition" {
			continue
		}

		err := build(state, child, content)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleNamedChildren(state *State, node *sitter.Node, content []byte) error {
	if node.NamedChildCount() <= 0 {
		return nil
	}

	for i := range node.NamedChildCount() {
		err := build(state, node.NamedChild(int(i)), content)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleProgram(state *State, node *sitter.Node, content []byte) error {
	cfg := &FunctionCFG{Blocks: []*Block{}, Node: node, Type: "program"}
	state.AllCfg = append(state.AllCfg, cfg)
	state.cfgStack.Push(cfg)

	start := state.cfg().AddBlock("Program start")
	end := state.cfg().AddBlock("Program end")

	state.cfg().Start = start
	state.cfg().End = end

	state.current = start

	for i := range node.NamedChildCount() {
		err := build(state, node.NamedChild(int(i)), content)
		if err != nil {
			return err
		}
	}

	state.cfg().AddEdge(state.current, end)

	return nil
}

func handleReturn(state *State, node *sitter.Node, content []byte) error {
	prevBlock := state.current
	returnBlock := state.cfg().AddBlock("Return block")
	afterReturnBlock := state.cfg().AddBlock("After return block")

	state.current = returnBlock

	state.AddInstruction(InstructionJump, "", node, "", content)

	state.cfg().AddEdge(prevBlock, returnBlock)
	state.cfg().AddEdge(returnBlock, state.cfg().End)

	state.current = afterReturnBlock

	return nil
}

func handleBreak(state *State, node *sitter.Node, content []byte) error {
	prevBlock := state.current
	afterBlock := state.peekBreakBlock()
	breakBlock := state.cfg().AddBlock("Break block")

	state.current = breakBlock

	state.AddInstruction(InstructionBranch, "", node, "", content)

	state.cfg().AddEdge(prevBlock, breakBlock)
	state.cfg().AddEdge(breakBlock, afterBlock)

	return nil
}

func handleContinue(state *State, node *sitter.Node, content []byte) error {
	prevBlock := state.current
	afterBlock := state.peekContinueBlock()
	breakBlock := state.cfg().AddBlock("Continue block")

	state.current = breakBlock

	state.AddInstruction(InstructionBranch, "", node, "", content)

	state.cfg().AddEdge(prevBlock, breakBlock)
	state.cfg().AddEdge(breakBlock, afterBlock)

	return nil
}

func handleCall(state *State, node *sitter.Node, content []byte) error {
	state.AddInstruction(InstructionCall, "", node, "", content)
	err := handleNamedChildren(state, node, content)
	if err != nil {
		return err
	}

	return nil
}

func handleFunction(state *State, node *sitter.Node, content []byte) error {
	blockName := "Function"

	nameContent := "function"
	name := node.ChildByFieldName("name")
	if name != nil {
		nameContent := name.Content(content)
		blockName = blockName + " " + nameContent
	}

	state.AddInstruction(InstructionAssign, nameContent, node, "", content)

	cfg := &FunctionCFG{Blocks: []*Block{}, Node: node, Type: "function"}
	state.AllCfg = append(state.AllCfg, cfg)
	state.cfgStack.Push(cfg)

	start := state.cfg().AddBlock(blockName + " start")
	end := state.cfg().AddBlock(blockName + " end")

	state.cfg().Start = start
	state.cfg().End = end

	body := node.ChildByFieldName("body")
	if body == nil {
		return nil
	}

	prevCurrent := state.current
	state.current = start

	err := build(state, body, content)
	if err != nil {
		return err
	}

	state.cfg().AddEdge(state.current, end)

	state.cfgStack.Pop()
	state.current = prevCurrent

	return nil
}

func handleIf(state *State, node *sitter.Node, content []byte) error {
	condBlock := state.cfg().AddBlock("If condition block")
	thenBlock := state.cfg().AddBlock("If then block")
	elseBlock := state.cfg().AddBlock("If else block")
	afterBlock := state.cfg().AddBlock("If after block")

	conditionNode := node.ChildByFieldName("condition")
	if conditionNode == nil {
		return errors.New("conditionNode unexpectedly nil")
	}

	state.AddInstruction(InstructionBranch, "", node, "", content)

	state.cfg().AddEdge(state.current, condBlock)
	condBlock.Node = conditionNode

	state.current = condBlock
	state.AddInstruction(InstructionBranch, "", node, "", content)

	if !condBlock.hasConstantFalse(content) {
		state.cfg().AddEdge(state.current, thenBlock)
	}

	thenNode := node.ChildByFieldName("consequence")
	if thenNode == nil {
		return errors.New("thenNode unexpectedly nil")
	}

	thenBlock.Node = thenNode

	elseNode := node.ChildByFieldName("alternative")
	if elseNode == nil {
		state.cfg().AddEdge(state.current, afterBlock)
	} else {
		if !condBlock.hasConstantTrue(content) {
			state.cfg().AddEdge(state.current, elseBlock)
		}

		elseBlock.Node = elseNode
	}

	state.current = thenBlock
	state.AddInstruction(InstructionBranch, "", thenBlock.Node, "", content)

	err := build(state, thenNode, content)
	if err != nil {
		return err
	}

	if len(state.current.After) == 0 {
		state.cfg().AddEdge(state.current, afterBlock)
	}

	if elseNode != nil {
		state.current = elseBlock
		err := build(state, elseNode, content)
		if err != nil {
			return err
		}

		if len(state.current.After) == 0 {
			state.cfg().AddEdge(state.current, afterBlock)
		}
	}

	state.current = afterBlock

	return nil
}

func handleWhile(state *State, node *sitter.Node, content []byte) error {
	condBlock := state.cfg().AddBlock("While condition block")
	bodyBlock := state.cfg().AddBlock("While body block")
	afterBlock := state.cfg().AddBlock("While after block")

	state.pushLoopBlocks(condBlock, afterBlock)

	conditiondNode := node.ChildByFieldName("condition")
	if conditiondNode == nil {
		return errors.New("conditiondNode unexpectedly nil")
	}

	state.AddInstruction(InstructionBranch, "", node, "", content)

	state.cfg().AddEdge(state.current, condBlock)
	condBlock.Node = conditiondNode

	state.current = condBlock
	state.AddInstruction(InstructionBranch, "", node, "", content)

	if !condBlock.hasConstantTrue(content) {
		state.cfg().AddEdge(condBlock, afterBlock)
	}

	if !condBlock.hasConstantFalse(content) {
		state.cfg().AddEdge(state.current, bodyBlock)
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return errors.New("bodyNode unexpectedly nil")
	}

	bodyBlock.Node = bodyNode
	state.current = bodyBlock
	err := build(state, bodyNode, content)
	if err != nil {
		return err
	}

	if len(state.current.After) == 0 {
		state.cfg().AddEdge(state.current, condBlock)
	}

	state.current = afterBlock

	state.popLoopBlocks()

	return nil
}

func handleForIn(state *State, node *sitter.Node, content []byte) error {
	state.AddInstruction(InstructionBranch, "", node, "", content)

	initialiseBlock := state.cfg().AddBlock("For-in initialisation block")
	nextBlock := state.cfg().AddBlock("For-in condition block")

	bodyBlock := state.cfg().AddBlock("For-in body block")
	afterBlock := state.cfg().AddBlock("For-in after block")

	state.pushLoopBlocks(nextBlock, afterBlock)

	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return errors.New("leftNode or rightNode unexpectedly nil")
	}

	state.cfg().AddEdge(state.current, initialiseBlock)

	state.current = initialiseBlock
	state.AddInstruction(InstructionAssign, "%iter", rightNode, rightNode.Content(content)+"[Symbol.iterator]()", content)

	state.cfg().AddEdge(initialiseBlock, nextBlock)
	state.current = nextBlock
	state.AddInstruction(InstructionAssign, "%value", rightNode, "%iter.next()", content)
	state.AddInstruction(InstructionBranch, "!%value.done", rightNode, "", content)

	state.cfg().AddEdge(nextBlock, bodyBlock)
	state.cfg().AddEdge(state.current, afterBlock)

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return errors.New("bodyNode unexpectedly nil")
	}

	bodyBlock.Node = bodyNode
	state.current = bodyBlock

	err := build(state, bodyNode, content)
	if err != nil {
		return err
	}

	if len(state.current.After) == 0 {
		// If the loop doesn't break/return
		state.cfg().AddEdge(state.current, nextBlock)
	}

	state.current = afterBlock

	state.popLoopBlocks()

	return nil
}

func handleVariableDeclaration(state *State, node *sitter.Node, content []byte) error {
	declarator := node.NamedChild(0)
	if declarator == nil || declarator.Type() != "variable_declarator" {
		return errors.New("declarator unexpectedly nil")
	}

	nameNode := declarator.ChildByFieldName("name")
	valueNode := declarator.ChildByFieldName("value")

	if valueNode == nil || nameNode == nil {
		return errors.New("valueNode or nameNode unexpectedly nil")
	}

	name := nameNode.Content(content)
	value := valueNode.Content(content)

	err := build(state, valueNode, content)
	if err != nil {
		return err
	}

	state.AddInstruction(InstructionAssign, name, node, value, content)

	return nil
}

func BuildGraphFromContent(content string) (*State, error) {
	c := []byte(content)
	state := newState(c)

	root, err := utils.ParseText(c, utils.TypeScript)
	if err != nil {
		return nil, err
	}

	err = build(state, root, c)
	if err != nil {
		return nil, err
	}

	return state, nil
}

func BuildGraphFromFile(file *parser.File) (*State, error) {
	content := file.Snapshot().Content

	return BuildGraphFromContent(content)
}

func (s *State) PrintFromState(sb *strings.Builder, visited *map[*Block]any) {
	sb.WriteString("digraph {\n")

	for _, cfg := range s.AllCfg {
		fmt.Fprintf(sb, "subgraph cluster_%p {\n", cfg)

		start := cfg.Start
		fmt.Fprintf(sb, "\"%p\" [label=\"%s (%d)\"]\n", start, start.Label, len(start.Instructions))

		for _, after := range start.After {
			printFromBlock(sb, visited, start, after)
		}

		fmt.Fprintf(sb, "}\n")
	}

	sb.WriteString("}\n")
}

func printFromBlock(sb *strings.Builder, visited *map[*Block]any, parent *Block, block *Block) {

	fmt.Fprintf(sb, "\"%p\" -> \"%p\"\n", parent, block)

	if _, found := (*visited)[block]; found {
		return
	}

	(*visited)[block] = true
	fmt.Fprintf(sb, "\"%p\" [label=\"%s (%d)\"]\n", block, block.Label, len(block.Instructions))

	for _, after := range block.After {
		printFromBlock(sb, visited, block, after)
	}
}

func checkBinaryExpression(node *sitter.Node, content []byte, check func(*sitter.Node, []byte) bool) bool {
	if node == nil || node.Type() != "binary_expression" {
		return false
	}

	operatorNode := node.ChildByFieldName("operator")
	if operatorNode == nil {
		return false
	}

	operator := operatorNode.Content(content)
	if operator != "&&" {
		return false
	}

	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return false
	}

	rightNode := node.ChildByFieldName("right")
	if rightNode == nil {
		return false
	}

	return check(leftNode, content) || check(rightNode, content)
}

func getExpressionNode(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}

	if node.Type() == "parenthesized_expression" {
		node = node.NamedChild(0)
		if node == nil {
			return nil
		}
	}

	return node
}

//nolint:unused
func hasConstantExpression(node *sitter.Node, content []byte) bool {
	n := getExpressionNode(node)

	if n == nil {
		return false
	}

	if n.Type() == "true" || n.Type() == "false" {
		return true
	}

	if n.Type() == "binary_expression" {
		return checkBinaryExpression(n, content, hasConstantExpression)
	}

	return false
}

func hasConstantFalse(node *sitter.Node, content []byte) bool {
	n := getExpressionNode(node)

	if n == nil {
		return false
	}

	if n.Type() == "false" {
		return true
	}

	if n.Type() == "binary_expression" {
		return checkBinaryExpression(n, content, hasConstantFalse)
	}

	return false
}

func hasConstantTrue(node *sitter.Node, content []byte) bool {
	n := getExpressionNode(node)

	if n == nil {
		return false
	}

	if n.Type() == "true" {
		return true
	}

	if n.Type() == "binary_expression" {
		return checkBinaryExpression(n, content, hasConstantTrue)
	}

	return false
}
