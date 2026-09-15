package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type identifier struct {
	commonNode
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

func visitIdentifier(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	identifier := identifier{commonNode: makeCommonNode("identifier", state, node)}

	return &identifier, nil
}
