package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type ExportStatement struct {
	commonNode
	Decorators  []*decorator
	Declaration nodeInterface
}

func (a *ExportStatement) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	for _, decorator := range a.Decorators {
		if decorator != nil && decorator.isUnderCursor(offset) {
			return decorator.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	if a.Declaration != nil && a.Declaration.isUnderCursor(offset) {
		return a.Declaration.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (a *ExportStatement) getNode() impl[nodeInterface] {
	commonNode, _ := a.Declaration.(impl[nodeInterface])
	return commonNode
}

func (a *ExportStatement) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, decorator := range a.Decorators {
		if decorator == nil {
			continue
		}

		if ret := decorator.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := a.Declaration.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitExportStatement(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	exportStatement := ExportStatement{commonNode: makeCommonNode("exportStatement", state, node)}
	exportStatement.setImpl(&exportStatement)

	decorators := []*decorator{}
	for _, decoratorNode := range node.ChildrenByFieldName("decorator", node.Walk()) {
		decoratorCommonNode, err := walk.VisitNode(&decoratorNode, state, 0, funcMap, false)
		if decoratorCommonNode == nil {
			return nil, err
		}

		var ddecorator *decorator
		if decorator, isIdentifier := IsNode[*decorator](decoratorCommonNode); isIdentifier {
			ddecorator = decorator
		} else {
			return nil, fmt.Errorf("invalid ast: decorator isn't a decorator")
		}

		decorators = append(decorators, ddecorator)
	}

	var declaration nodeInterface
	var err error
	declarationNode := node.ChildByFieldName("declaration")
	if declarationNode != nil {
		declaration, err = walk.VisitNode(declarationNode, state, 0, funcMap, false)
		if declaration == nil {
			return nil, err
		}
	}

	exportStatement.Declaration = declaration
	exportStatement.Decorators = decorators

	return &exportStatement, nil
}
