package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type classHeritage struct {
	commonNode
	Extends    *extendsClause
	Implements *implementsClause
}

func (t *classHeritage) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !t.isUnderCursor(offset) {
		return nil, false
	}

	if first && t.getKind() == kind {
		return t, true
	}

	if t.Extends != nil && t.Extends.isUnderCursor(offset) {
		return t.Extends.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if t.Implements != nil && t.Implements.isUnderCursor(offset) {
		return t.Implements.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || t.getKind() == kind {
		return t, true
	}

	return nil, false
}

func (t *classHeritage) visit(exec func(nodeInterface) int) int {
	if ret := exec(t); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if t.Extends != nil {
		if ret := t.Extends.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if t.Implements != nil {
		if ret := t.Implements.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitClassHeritage(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	classHeritage := classHeritage{commonNode: makeCommonNode("classHeritage", state, node)}
	classHeritage.setImpl(&classHeritage)

	var eextendsClause *extendsClause
	var iimplementsClause *implementsClause

	for _, child := range node.NamedChildren(node.Walk()) {
		if child.Kind() == "extends_clause" {
			extendsClauseCommonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
			if err != nil {
				return nil, err
			}

			if extendsClause, isExtendsClause := IsNode[*extendsClause](extendsClauseCommonNode); isExtendsClause {
				eextendsClause = extendsClause
			} else {
				return nil, fmt.Errorf("invalid ast: extends clause isn't an extends clause")
			}

			continue
		}

		if child.Kind() == "implements_clause" {
			implementsClauseCommonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
			if err != nil {
				return nil, err
			}

			if implementsClause, isImplementsClause := IsNode[*implementsClause](implementsClauseCommonNode); isImplementsClause {
				iimplementsClause = implementsClause
			} else {
				return nil, fmt.Errorf("invalid ast: implements clause isn't an implements clause")
			}

			continue
		}
	}

	classHeritage.Extends = eextendsClause
	classHeritage.Implements = iimplementsClause

	return &classHeritage, nil
}
