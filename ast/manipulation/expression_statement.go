package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type expressionStatement struct {
	commonNode
	Children []nodeInterface
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

func visitExpressionStatement(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
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
