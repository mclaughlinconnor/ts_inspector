package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type typeParameter struct {
	commonNode
	isConst    bool
	Name       nodeInterface
	Constraint nodeInterface
	Value      nodeInterface
}

func (t *typeParameter) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !t.isUnderCursor(offset) {
		return nil, false
	}

	if first && t.getKind() == kind {
		return t, true
	}

	if t.Name.isUnderCursor(offset) {
		return t.Name.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if t.Constraint.isUnderCursor(offset) {
		return t.Constraint.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if t.Value.isUnderCursor(offset) {
		return t.Value.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || t.getKind() == kind {
		return t, true
	}

	return nil, false
}

func (t *typeParameter) visit(exec func(nodeInterface) int) int {
	if ret := exec(t); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := t.Name.visit(exec); ret == VisitAbort {
		return ret
	}

	if t.Constraint != nil {
		if ret := t.Constraint.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if t.Value != nil {
		if ret := t.Value.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitTypeParameter(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	typeParameter := typeParameter{commonNode: makeCommonNode("typeParameter", state, node)}
	typeParameter.setImpl(&typeParameter)

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, newAstErrorTS(node, state, "missing name")
	}

	constraintNode := node.ChildByFieldName("constraint")
	valueNode := node.ChildByFieldName("value")

	name, err := walk.VisitNode(nameNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	var constraint nodeInterface
	if constraintNode != nil {
		constraint, err = walk.VisitNode(constraintNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	var value nodeInterface
	if valueNode != nil {
		value, err = walk.VisitNode(valueNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	isConstant := node.Child(0).Kind() == "const"

	typeParameter.Constraint = constraint
	typeParameter.isConst = isConstant
	typeParameter.Name = name
	typeParameter.Value = value

	return &typeParameter, nil
}
