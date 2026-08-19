package cfg

import (
	"fmt"
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
	Start  *Block
	End    *Block
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

func (s *State) popBreakBlock() *Block {
	s.continueStack.Pop()
	return *s.breakStack.Pop()
}

func (s *State) popContinueBlock() *Block {
	s.breakStack.Pop()
	return *s.continueStack.Pop()
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

func (state *State) AddInstruction(kind InstructionKind, left string, node *sitter.Node, right string, content []byte) {
	if len(state.current.After) != 0 {
		current := state.cfg().AddBlock("Continuation")
		state.current = current
	}

	instruction := Instruction{kind, left, node, right, node.Content(content)}
	state.current.Instructions = append(state.current.Instructions, &instruction)
}

type visitorFunction = func(state *State, node *sitter.Node, content []byte)

var visitMap = map[string]visitorFunction{
	"arrow_function":       handleFunction,
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
	"method_definition":    handleFunction,
	"program":              handleProgram,
	"return_statement":     handleReturn,
	"statement_block":      handleNamedChildren,
	"variable_declaration": handleVariableDeclaration,
	"while_statement":      handleWhile,
}

var funcMap = walk.NewVisitorFuncsMap[*State]()

func InitBuilder() {
	for k, v := range visitMap {
		funcMap[k] = func(node *sitter.Node, state *State, indexInParent int, funcMap walk.VisitorFuncMap[*State]) *State {
			v(state, node, state.content)

			return state
		}
	}
}

func build(state *State, root *sitter.Node, content []byte) {
	walk.WalkTypeScriptShallow(root, state, funcMap)
}

func handleClass(state *State, node *sitter.Node, content []byte) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}

	for i := range body.NamedChildCount() {
		child := body.NamedChild(int(i))
		if child.Type() != "method_definition" {
			continue
		}

		build(state, child, content)
	}
}

func handleNamedChildren(state *State, node *sitter.Node, content []byte) {
	if node.NamedChildCount() <= 0 {
		return
	}

	for i := range node.NamedChildCount() {
		build(state, node.NamedChild(int(i)), content)
	}
}

func handleProgram(state *State, node *sitter.Node, content []byte) {
	cfg := &FunctionCFG{Blocks: []*Block{}}
	state.AllCfg = append(state.AllCfg, cfg)
	state.cfgStack.Push(cfg)

	start := state.cfg().AddBlock("Program start")
	end := state.cfg().AddBlock("Program end")

	state.cfg().Start = start
	state.cfg().End = end

	state.current = start

	for i := range node.NamedChildCount() {
		build(state, node.NamedChild(int(i)), content)
	}

	state.cfg().AddEdge(state.current, end)
}

func handleReturn(state *State, node *sitter.Node, content []byte) {
	prevBlock := state.current
	returnBlock := state.cfg().AddBlock("Return block")
	afterReturnBlock := state.cfg().AddBlock("After return block")

	state.current = returnBlock

	state.AddInstruction(InstructionJump, "", node, "", content)

	state.cfg().AddEdge(prevBlock, returnBlock)
	state.cfg().AddEdge(returnBlock, state.cfg().End)

	state.current = afterReturnBlock
}

func handleBreak(state *State, node *sitter.Node, content []byte) {
	prevBlock := state.current
	afterBlock := state.peekBreakBlock()
	breakBlock := state.cfg().AddBlock("Break block")

	state.current = breakBlock

	state.AddInstruction(InstructionBranch, "", node, "", content)

	state.cfg().AddEdge(prevBlock, breakBlock)
	state.cfg().AddEdge(breakBlock, afterBlock)
}

func handleContinue(state *State, node *sitter.Node, content []byte) {
	prevBlock := state.current
	afterBlock := state.peekContinueBlock()
	breakBlock := state.cfg().AddBlock("Continue block")

	state.current = breakBlock

	state.AddInstruction(InstructionBranch, "", node, "", content)

	state.cfg().AddEdge(prevBlock, breakBlock)
	state.cfg().AddEdge(breakBlock, afterBlock)
}

func handleCall(state *State, node *sitter.Node, content []byte) {
	state.AddInstruction(InstructionCall, "", node, "", content)
}

func handleFunction(state *State, node *sitter.Node, content []byte) {
	blockName := "Function"

	nameContent := "function"
	name := node.ChildByFieldName("name")
	if name != nil {
		nameContent := name.Content(content)
		blockName = blockName + " " + nameContent
	}

	state.AddInstruction(InstructionAssign, nameContent, node, "", content)

	cfg := &FunctionCFG{Blocks: []*Block{}}
	state.AllCfg = append(state.AllCfg, cfg)
	state.cfgStack.Push(cfg)

	start := state.cfg().AddBlock(blockName + " start")
	end := state.cfg().AddBlock(blockName + " end")

	state.cfg().Start = start
	state.cfg().End = end

	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}

	prevCurrent := state.current
	state.current = start

	build(state, body, content)

	state.cfg().AddEdge(state.current, end)

	state.cfgStack.Pop()
	state.current = prevCurrent
}

func handleIf(state *State, node *sitter.Node, content []byte) {
	condBlock := state.cfg().AddBlock("If condition block")
	thenBlock := state.cfg().AddBlock("If then block")
	elseBlock := state.cfg().AddBlock("If else block")
	afterBlock := state.cfg().AddBlock("If after block")

	conditionNode := node.ChildByFieldName("condition")
	if conditionNode == nil {
		return
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
		return // grammar guarantees that it exists
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

	build(state, thenNode, content)
	if len(state.current.After) == 0 {
		state.cfg().AddEdge(state.current, afterBlock)
	}

	if elseNode != nil {
		state.current = elseBlock
		build(state, elseNode, content)

		if len(state.current.After) == 0 {
			state.cfg().AddEdge(state.current, afterBlock)
		}
	}

	state.current = afterBlock
}

func handleWhile(state *State, node *sitter.Node, content []byte) {
	condBlock := state.cfg().AddBlock("While condition block")
	bodyBlock := state.cfg().AddBlock("While body block")
	afterBlock := state.cfg().AddBlock("While after block")

	state.pushLoopBlocks(condBlock, afterBlock)

	conditiondNode := node.ChildByFieldName("condition")
	if conditiondNode == nil {
		return
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
		return // grammar guarantees that it exists
	}

	bodyBlock.Node = bodyNode
	state.current = bodyBlock
	build(state, bodyNode, content)
	if len(state.current.After) == 0 {
		state.cfg().AddEdge(state.current, condBlock)
	}

	state.current = afterBlock

	state.popLoopBlocks()
}

func handleForIn(state *State, node *sitter.Node, content []byte) {
	state.AddInstruction(InstructionBranch, "", node, "", content)

	initialiseBlock := state.cfg().AddBlock("For-in initialisation block")
	nextBlock := state.cfg().AddBlock("For-in condition block")

	bodyBlock := state.cfg().AddBlock("For-in body block")
	afterBlock := state.cfg().AddBlock("For-in after block")

	state.pushLoopBlocks(nextBlock, afterBlock)

	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return
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
		return
	}

	bodyBlock.Node = bodyNode
	state.current = bodyBlock

	build(state, bodyNode, content)

	if len(state.current.After) != 0 {
		skipsAfterBlock := !slices.Contains(state.current.After, afterBlock)
		if !skipsAfterBlock {
			state.cfg().AddEdge(state.current, afterBlock)
		}
	}

	state.current = afterBlock

	state.popLoopBlocks()
}

func handleVariableDeclaration(state *State, node *sitter.Node, content []byte) {
	declarator := node.NamedChild(0)
	if declarator == nil || declarator.Type() != "variable_declarator" {
		return
	}

	nameNode := declarator.ChildByFieldName("name")
	valueNode := declarator.ChildByFieldName("value")

	if valueNode == nil || nameNode == nil {
		return
	}

	name := nameNode.Content(content)
	value := valueNode.Content(content)

	state.AddInstruction(InstructionAssign, name, node, value, content)
}

func Run() {
	content := []byte("function hello() { for (const x of xs) { break } op(); } function hello() { for (const x of xs) { continue } op(); } function hello() { for (const x of xs) { return } op(); }")

	state := newState([]byte(content))

	root, err := utils.ParseText(content, utils.TypeScript)
	if err != nil {
		panic(err)
	}

	for i := range root.NamedChildCount() { // the root it a `(program)`
		build(state, root.NamedChild(int(i)), content)
	}

	sb := strings.Builder{}
	visited := map[*Block]any{}
	state.PrintFromState(&sb, &visited)

	println(sb.String())
}

func BuildGraphFromContent(content string) (*State, error) {
	c := []byte(content)
	state := newState(c)

	root, err := utils.ParseText(c, utils.TypeScript)
	if err != nil {
		return nil, err
	}

	build(state, root, c)

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
	if node.Type() != "binary_expression" {
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

func hasConstantExpression(node *sitter.Node, content []byte) bool {
	n := getExpressionNode(node)

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

	if n.Type() == "true" {
		return true
	}

	if n.Type() == "binary_expression" {
		return checkBinaryExpression(n, content, hasConstantTrue)
	}

	return false
}
