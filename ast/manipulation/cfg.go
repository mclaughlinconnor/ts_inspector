package manipulation

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"ts_inspector/utils"
)

type Cfg struct {
	allCfg        []*FunctionCfg
	breakStack    utils.Stack[*CfgBlock]
	cfgStack      utils.Stack[*FunctionCfg]
	continueStack utils.Stack[*CfgBlock]
	current       *CfgBlock
}

type CfgBlock struct {
	After        []*CfgBlock
	Before       []*CfgBlock
	Instructions []*instruction
	Label        string
	Node         nodeInterface
}

type FunctionCfg struct {
	End   *CfgBlock
	Kind  string
	Node  nodeInterface
	Start *CfgBlock

	blocks []*CfgBlock
}

type instructionKind = int

const (
	instructionJump instructionKind = iota
	instructionBranch
	instructionCall
	instructionAssign
)

type instruction struct {
	Node nodeInterface

	kind  instructionKind
	left  string
	right string
	text  string
}

func (c *Cfg) GetAllFunctionCfg() []*FunctionCfg {
	return c.allCfg
}

func (c *Cfg) addInstruction(kind instructionKind, left string, node nodeInterface, right string) {
	if len(c.current.After) != 0 {
		current := c.currentCfg().addBlock("Continuation")
		c.current = current
	}

	instruction := instruction{node, kind, left, right, node.getText()}
	c.current.Instructions = append(c.current.Instructions, &instruction)
}

func (c *Cfg) currentCfg() *FunctionCfg {
	return *c.cfgStack.Peek()
}

func (c *Cfg) pushLoopBlocks(continueBlock *CfgBlock, breakBlock *CfgBlock) {
	c.continueStack.Push(continueBlock)
	c.breakStack.Push(breakBlock)
}

func (c *Cfg) peekBreakBlock() *CfgBlock {
	b := c.breakStack.Peek()
	if b != nil {
		return *b
	}

	return nil
}

func (c *Cfg) peekContinueBlock() *CfgBlock {
	b := c.continueStack.Peek()
	if b != nil {
		return *b
	}

	return nil
}

func (c *Cfg) popLoopBlocks() {
	c.continueStack.Pop()
	c.breakStack.Pop()
}

func (c *CfgBlock) countDownwardEdges(seen map[*CfgBlock]bool) int {
	seen[c] = true
	count := len(c.After)

	for _, a := range c.After {
		if seen[a] {
			continue
		}

		count += a.countDownwardEdges(seen)
	}

	return count
}

func (c *CfgBlock) getDownwardNodes(blocks map[*CfgBlock]bool) {
	blocks[c] = true

	for _, a := range c.After {
		if blocks[a] {
			continue
		}

		a.getDownwardNodes(blocks)
	}
}

//nolint:unused
func (c *CfgBlock) hasConstantExpression() bool {
	return c.Node.getNode().getImpl().hasConstantExpression()
}

func (c *CfgBlock) hasConstantFalse() bool {
	return c.Node.getNode().getImpl().hasConstantFalse()
}

func (c *CfgBlock) hasConstantTrue() bool {
	return c.Node.getNode().getImpl().hasConstantTrue()
}

func (c *CfgBlock) CalculateCyclomaticComplexity() int {
	return c.CountDownwardEdges() - c.CountDownwardNodes() + 2
}

func (c *CfgBlock) CountDownwardEdges() int {
	seen := map[*CfgBlock]bool{}

	return c.countDownwardEdges(seen)
}

func (c *CfgBlock) CountDownwardNodes() int {
	blocks := map[*CfgBlock]bool{}
	c.getDownwardNodes(blocks)

	return len(slices.Collect(maps.Keys(blocks)))
}

func (s *Cfg) PrintFromState(sb *strings.Builder, visited *map[*CfgBlock]any) {
	sb.WriteString("digraph {\n")

	for _, cfg := range s.GetAllFunctionCfg() {
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

func (f *FunctionCfg) GetBlocks() []*CfgBlock {
	return f.blocks
}

func (f *FunctionCfg) addEdge(from *CfgBlock, to *CfgBlock) {
	from.After = append(from.After, to)
	to.Before = append(to.Before, from)
}

func (f *FunctionCfg) addBlock(label string) *CfgBlock {
	block := CfgBlock{Label: label}
	f.blocks = append(f.blocks, &block)

	return &block
}

func (f *FunctionCfg) CalculateCyclomaticComplexity() int {
	return f.Start.CalculateCyclomaticComplexity()
}

func (f *FunctionCfg) CountDownwardEdges() int {
	return f.Start.CountDownwardEdges()
}

func (f *FunctionCfg) CountDownwardNodes() int {
	return f.Start.CountDownwardNodes()
}

func newCfg() *Cfg {
	continueStack := utils.NewStack[*CfgBlock]()
	cfgStack := utils.NewStack[*FunctionCfg]()

	return &Cfg{
		allCfg:        []*FunctionCfg{},
		continueStack: *continueStack,
		cfgStack:      *cfgStack,
	}
}

func printFromBlock(sb *strings.Builder, visited *map[*CfgBlock]any, parent *CfgBlock, block *CfgBlock) {
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
