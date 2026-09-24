package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type program struct {
	childedCommonNode
}

func (p *program) _buildCfgBlock() (bool, error) {
	currentCfg := &FunctionCfg{blocks: []*CfgBlock{}, Node: p, Kind: "program"}
	cfg := p.getCfg()

	cfg.allCfg = append(cfg.allCfg, currentCfg)
	cfg.cfgStack.Push(currentCfg)

	start := cfg.currentCfg().addBlock("Program start")
	end := cfg.currentCfg().addBlock("Program end")

	cfg.currentCfg().Start = start
	cfg.currentCfg().End = end

	cfg.current = start

	var outsideError error

	p.visitChildren(func(ni nodeInterface) int {
		_, err := ni.buildCfgBlock()
		if err == nil {
			outsideError = err
			return VisitAbort
		}

		return VisitSkip
	})

	if outsideError != nil {
		return true, outsideError
	}

	cfg.currentCfg().addEdge(cfg.current, end)

	return false, nil
}

func visitProgram(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	program := program{childedCommonNode: makeCommonChildedNode("program", state, node)}
	program.setImpl(&program)

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

	program.children = children

	return &program, nil
}
