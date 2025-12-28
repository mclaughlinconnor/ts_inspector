package cfg

import (
	"fmt"
	"strings"
	"ts_inspector/ast/walk"
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
	node  *sitter.Node
	right string
	text  string
}

type Block struct {
	node         *sitter.Node
	instructions []*Instruction
	before       []*Block
	after        []*Block
	label        string
}

type FunctionCFG struct {
	blocks []*Block
	start  *Block
	end    *Block
}

type State struct {
	allCfg        []*FunctionCFG
	breakStack    utils.Stack[*Block]
	cfg           *FunctionCFG
	continueStack utils.Stack[*Block]
	current       *Block
}

func (s *State) pushLoopBlocks(continueBlock *Block, breakBlock *Block) {
	s.continueStack.Push(continueBlock)
	s.breakStack.Push(breakBlock)
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
	from.after = append(from.after, to)
	to.before = append(to.before, from)
}

func (cfg *FunctionCFG) AddBlock(label string) *Block {
	block := Block{label: label}
	cfg.blocks = append(cfg.blocks, &block)

	return &block
}

func (b *Block) AddInstruction(kind InstructionKind, left string, node *sitter.Node, right string, content []byte) {
	instruction := Instruction{kind, left, node, right, node.Content(content)}
	b.instructions = append(b.instructions, &instruction)
}

func build(state *State, root *sitter.Node, content []byte) {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["return_statement"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleReturn(state, node, content)

		return nil
	}

	funcMap["break_statement"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleBreak(state, node, content)

		return nil
	}

	funcMap["continue_statement"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleContinue(state, node, content)

		return nil
	}

	funcMap["function_declaration"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleFunction(state, node, content)

		return nil
	}

	funcMap["statement_block"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		if node.NamedChildCount() <= 0 {
			return nil
		}

		walk.VisitNamedChildren(node, nil, funcMap, true)

		return nil
	}

	funcMap["expression_statement"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		if node.NamedChildCount() <= 0 {
			return nil
		}

		walk.VisitNamedChildren(node, nil, funcMap, true)

		return nil
	}

	funcMap["else_clause"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		if node.NamedChildCount() <= 0 {
			return nil
		}

		walk.VisitNamedChildren(node, nil, funcMap, true)

		return nil
	}

	funcMap["call_expression"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleCall(state, node, content)

		return nil
	}

	funcMap["if_statement"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleIf(state, node, content)

		return nil
	}

	funcMap["while_statement"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleWhile(state, node, content)

		return nil
	}

	funcMap["for_in_statement"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleForIn(state, node, content)

		return nil
	}

	walk.WalkTypeScriptShallow(root, nil, funcMap)
}

func handleReturn(state *State, _ *sitter.Node, _ []byte) {
	returnBlock := state.cfg.AddBlock("Return block")

	state.cfg.AddEdge(state.current, returnBlock)
	state.cfg.AddEdge(returnBlock, state.cfg.end)

	state.current = returnBlock
}

func handleBreak(state *State, _ *sitter.Node, _ []byte) {
	afterBlock := state.popBreakBlock()
	breakBlock := state.cfg.AddBlock("Break block")

	state.cfg.AddEdge(state.current, breakBlock)
	state.cfg.AddEdge(breakBlock, afterBlock)

	state.current = breakBlock
}

func handleContinue(state *State, _ *sitter.Node, _ []byte) {
	afterBlock := state.popContinueBlock()
	breakBlock := state.cfg.AddBlock("Continue block")

	state.cfg.AddEdge(state.current, breakBlock)
	state.cfg.AddEdge(breakBlock, afterBlock)

	state.current = breakBlock
}

func handleCall(state *State, node *sitter.Node, content []byte) {
	state.current.AddInstruction(InstructionCall, "", node, "", content)
}

func handleFunction(state *State, node *sitter.Node, content []byte) {
	state.cfg = &FunctionCFG{blocks: []*Block{}}
	state.allCfg = append(state.allCfg, state.cfg)

	start := state.cfg.AddBlock("Function start")
	end := state.cfg.AddBlock("Function end")

	state.cfg.start = start
	state.cfg.end = end

	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}

	state.current = start

	build(state, body, content)

	state.cfg.AddEdge(state.current, end)
}

func handleIf(state *State, node *sitter.Node, content []byte) {
	condBlock := state.cfg.AddBlock("If condition block")
	thenBlock := state.cfg.AddBlock("If then block")
	elseBlock := state.cfg.AddBlock("If else block")
	afterBlock := state.cfg.AddBlock("If after block")

	conditiondNode := node.ChildByFieldName("condition")
	if conditiondNode == nil {
		return
	}

	condBlock.node = conditiondNode
	state.current.AddInstruction(InstructionBranch, "", node, "", content)
	state.cfg.AddEdge(state.current, condBlock)
	state.current = condBlock

	state.cfg.AddEdge(state.current, thenBlock)

	thenNode := node.ChildByFieldName("consequence")
	if thenNode == nil {
		return // grammar guarantees that it exists
	}

	thenBlock.node = thenNode

	elseNode := node.ChildByFieldName("alternative")
	if elseNode == nil {
		state.cfg.AddEdge(state.current, afterBlock)
	} else {
		state.cfg.AddEdge(state.current, elseBlock)
		elseBlock.node = elseNode
	}

	state.current = thenBlock
	build(state, thenNode, content)
	if len(state.current.after) == 0 {
		state.cfg.AddEdge(state.current, afterBlock)
	}

	if elseNode != nil {
		state.current = elseBlock
		build(state, elseNode, content)

		if len(state.current.after) == 0 {
			state.cfg.AddEdge(state.current, afterBlock)
		}
	}

	state.current = afterBlock
}

func handleWhile(state *State, node *sitter.Node, content []byte) {
	bodyBlock := state.cfg.AddBlock("While body block")
	condBlock := state.cfg.AddBlock("While condition block")
	afterBlock := state.cfg.AddBlock("While after block")

	state.pushLoopBlocks(condBlock, afterBlock)

	conditiondNode := node.ChildByFieldName("condition")
	if conditiondNode == nil {
		return
	}

	condBlock.node = conditiondNode
	state.current.AddInstruction(InstructionBranch, "", node, "", content)
	state.cfg.AddEdge(state.current, condBlock)
	state.cfg.AddEdge(condBlock, afterBlock)
	state.current = condBlock

	state.cfg.AddEdge(state.current, bodyBlock)

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return // grammar guarantees that it exists
	}

	bodyBlock.node = bodyNode
	state.current = bodyBlock
	build(state, bodyNode, content)
	if len(state.current.after) == 0 {
		state.cfg.AddEdge(state.current, condBlock)
	}

	state.current = afterBlock

	state.popLoopBlocks()
}

func handleForIn(state *State, node *sitter.Node, content []byte) {
	initialiseBlock := state.cfg.AddBlock("For-in initialisation block")
	nextBlock := state.cfg.AddBlock("For-in condition block")

	bodyBlock := state.cfg.AddBlock("For-in body block")
	afterBlock := state.cfg.AddBlock("For-in after block")

	state.pushLoopBlocks(nextBlock, afterBlock)

	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return
	}

	state.cfg.AddEdge(state.current, initialiseBlock)

	initialiseBlock.AddInstruction(InstructionAssign, "%iter", rightNode, rightNode.Content(content)+"[Symbol.iterator]()", content)

	state.cfg.AddEdge(initialiseBlock, nextBlock)
	nextBlock.AddInstruction(InstructionAssign, "%value", rightNode, "%iter.next()", content)
	nextBlock.AddInstruction(InstructionBranch, "!%value.done", rightNode, "", content)

	state.cfg.AddEdge(nextBlock, bodyBlock)
	state.cfg.AddEdge(nextBlock, afterBlock)

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return
	}

	bodyBlock.node = bodyNode
	state.current = bodyBlock

	build(state, bodyNode, content)

	if len(state.current.after) == 0 {
		state.cfg.AddEdge(state.current, nextBlock)
	}

	state.current = afterBlock

	state.popLoopBlocks()
}

func Run() {
	content := "function hello() { while (true) { ops(); opt(); if (cond) {continue} opt() } opt(); } function hello() { while (true) { ops(); opt(); if (cond) {continue} opt() } opt(); }"

	state := State{allCfg: []*FunctionCFG{}}

	utils.ParseFile(false, content, utils.TypeScript, nil, func(root *sitter.Node, content []byte, _ any) (any, error) {
		for i := range root.NamedChildCount() { // the root it a `(program)`
			build(&state, root.NamedChild(int(i)), content)
		}

		return nil, nil
	})

	sb := strings.Builder{}
	visited := map[*Block]any{}
	printFromState(&sb, &visited, &state)

	println(sb.String())
}

func printFromState(sb *strings.Builder, visited *map[*Block]any, state *State) {
	sb.WriteString("digraph {\n")

	for _, cfg := range state.allCfg {
		fmt.Fprintf(sb, "subgraph cluster_%p {\n", cfg)

		start := cfg.start
		fmt.Fprintf(sb, "\"%p\" [label=\"%s (%d)\"]\n", start, start.label, len(start.instructions))

		for _, after := range start.after {
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
	fmt.Fprintf(sb, "\"%p\" [label=\"%s (%d)\"]\n", block, block.label, len(block.instructions))

	for _, after := range block.after {
		printFromBlock(sb, visited, block, after)
	}
}
