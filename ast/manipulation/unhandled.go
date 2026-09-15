package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type unhandled struct {
	commonNode
	Children []nodeInterface
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

func visitUnhandled(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
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
