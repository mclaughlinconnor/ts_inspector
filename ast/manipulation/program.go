package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type program struct {
	commonNode
	Children []nodeInterface
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

func visitProgram(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
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
