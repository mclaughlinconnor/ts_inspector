package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type decorator struct {
	commonNode
	body nodeInterface
}

func (d *decorator) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !d.isUnderCursor(offset) {
		return nil, false
	}

	if first && d.getKind() == kind {
		return d, true
	}

	if d.body != nil && d.body.isUnderCursor(offset) {
		return d.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || d.getKind() == kind {
		return d, true
	}

	return nil, false
}

func (d *decorator) visit(exec func(nodeInterface) int) int {
	if ret := exec(d); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := d.body.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitDecorator(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	decorator := decorator{commonNode: makeCommonNode("decorator", state, node)}
	decorator.setImpl(&decorator)

	bodyNode := node.NamedChild(0)
	if bodyNode == nil {
		return nil, newAstErrorTS(node, state, "no decorator node")
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	decorator.body = body

	return &decorator, nil
}
