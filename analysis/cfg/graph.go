package cfg

import (
	"fmt"
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
	AllCfg        []*FunctionCFG
	breakStack    utils.Stack[*Block]
	cfgStack      utils.Stack[*FunctionCFG]
	continueStack utils.Stack[*Block]
	current       *Block
}

func newState() *State {
	return &State{AllCfg: []*FunctionCFG{}}
}

func (s *State) cfg() *FunctionCFG {
	return *s.cfgStack.Peek()
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

func build(state *State, root *sitter.Node, content []byte) {
	funcMap := walk.NewVisitorFuncsMap[any]()

	funcMap["program"] = func(node *sitter.Node, _ any, indexInParent int, funcMap walk.VisitorFuncMap[any]) any {
		handleProgram(state, node, content)

		return nil
	}

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

	state.current = returnBlock

	state.AddInstruction(InstructionJump, "", node, "", content)

	state.cfg().AddEdge(prevBlock, returnBlock)
	state.cfg().AddEdge(returnBlock, state.cfg().End)
}

func handleBreak(state *State, node *sitter.Node, content []byte) {
	prevBlock := state.current
	afterBlock := state.popBreakBlock()
	breakBlock := state.cfg().AddBlock("Break block")

	state.current = breakBlock

	state.AddInstruction(InstructionBranch, "", node, "", content)

	state.cfg().AddEdge(prevBlock, breakBlock)
	state.cfg().AddEdge(breakBlock, afterBlock)
}

func handleContinue(state *State, node *sitter.Node, content []byte) {
	prevBlock := state.current
	afterBlock := state.popContinueBlock()
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
	cfg := &FunctionCFG{Blocks: []*Block{}}
	state.AllCfg = append(state.AllCfg, cfg)
	state.cfgStack.Push(cfg)

	start := state.cfg().AddBlock("Function start")
	end := state.cfg().AddBlock("Function end")

	state.cfg().Start = start
	state.cfg().End = end

	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}

	state.current = start

	build(state, body, content)

	state.cfg().AddEdge(state.current, end)
}

func handleIf(state *State, node *sitter.Node, content []byte) {
	condBlock := state.cfg().AddBlock("If condition block")
	thenBlock := state.cfg().AddBlock("If then block")
	elseBlock := state.cfg().AddBlock("If else block")
	afterBlock := state.cfg().AddBlock("If after block")

	conditiondNode := node.ChildByFieldName("condition")
	if conditiondNode == nil {
		return
	}

	state.cfg().AddEdge(state.current, condBlock)
	condBlock.Node = conditiondNode

	state.current = condBlock
	state.AddInstruction(InstructionBranch, "", node, "", content)

	state.cfg().AddEdge(state.current, thenBlock)

	thenNode := node.ChildByFieldName("consequence")
	if thenNode == nil {
		return // grammar guarantees that it exists
	}

	thenBlock.Node = thenNode

	elseNode := node.ChildByFieldName("alternative")
	if elseNode == nil {
		state.cfg().AddEdge(state.current, afterBlock)
	} else {
		state.cfg().AddEdge(state.current, elseBlock)
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
	bodyBlock := state.cfg().AddBlock("While body block")
	condBlock := state.cfg().AddBlock("While condition block")
	afterBlock := state.cfg().AddBlock("While after block")

	state.pushLoopBlocks(condBlock, afterBlock)

	conditiondNode := node.ChildByFieldName("condition")
	if conditiondNode == nil {
		return
	}

	state.cfg().AddEdge(state.current, condBlock)
	condBlock.Node = conditiondNode

	state.current = condBlock
	state.AddInstruction(InstructionBranch, "", node, "", content)
	state.cfg().AddEdge(condBlock, afterBlock)

	state.cfg().AddEdge(state.current, bodyBlock)

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

	if len(state.current.After) == 0 {
		skipsAfterBlock := true
		for _, after := range state.current.After {
			if after == afterBlock {
				skipsAfterBlock = false
				break
			}
		}

		if !skipsAfterBlock {
			state.cfg().AddEdge(state.current, afterBlock)
		}
	}

	state.current = afterBlock

	state.popLoopBlocks()
}

func Run() {
	content := "function hello() { for (const x of xs) { break } op(); } function hello() { for (const x of xs) { continue } op(); } function hello() { for (const x of xs) { return } op(); }"

	state := newState()

	utils.ParseFile(false, content, utils.TypeScript, nil, func(root *sitter.Node, content []byte, _ any) (any, error) {
		for i := range root.NamedChildCount() { // the root it a `(program)`
			build(state, root.NamedChild(int(i)), content)
		}

		return nil, nil
	})

	sb := strings.Builder{}
	visited := map[*Block]any{}
	printFromState(&sb, &visited, state)

	println(sb.String())
}

func BuildGraph(file *parser.File) *State {
	content := file.Snapshot().Content

	state := newState()

	utils.ParseFile(false, content, utils.TypeScript, nil, func(root *sitter.Node, content []byte, _ any) (any, error) {
		build(state, root, content)

		return nil, nil
	})

	return state
}

func printFromState(sb *strings.Builder, visited *map[*Block]any, state *State) {
	sb.WriteString("digraph {\n")

	for _, cfg := range state.AllCfg {
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
