package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type pair struct {
	commonNode
	key   nodeInterface
	value nodeInterface
}

func (i *pair) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !i.isUnderCursor(offset) {
		return nil, false
	}

	if first && i.getKind() == kind {
		return i, true
	}

	if i.key.isUnderCursor(offset) {
		return i.key.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if i.value.isUnderCursor(offset) {
		return i.value.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || i.getKind() == kind {
		return i, true
	}

	return nil, false
}

func (i *pair) visit(exec func(nodeInterface) int) int {
	if ret := exec(i); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := i.key.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := i.value.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitPair(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	pair := pair{commonNode: makeCommonNode("pair", state, node)}
	pair.setImpl(&pair)

	keyNode := node.ChildByFieldName("key")
	if keyNode == nil {
		return nil, newAstErrorTS(node, state, "missing key")
	}

	valueNode := node.ChildByFieldName("value")
	if valueNode == nil {
		return nil, newAstErrorTS(node, state, "missing value")
	}

	key, err := walk.VisitNode(keyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	value, err := walk.VisitNode(valueNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	pair.key = key
	pair.value = value

	return &pair, nil
}
