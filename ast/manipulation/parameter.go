package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type parameter struct {
	commonNode
	accessibilityModifier *accessibilityModifier
	decorators            []*decorator
	isOptional            bool
	isOverride            bool
	isReadonly            bool
	pattern               nodeInterface
	ttype                 nodeInterface
	value                 nodeInterface
}

func (t *parameter) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !t.isUnderCursor(offset) {
		return nil, false
	}

	if first && t.getKind() == kind {
		return t, true
	}

	for _, decorator := range t.decorators {
		if decorator.isUnderCursor(offset) {
			return decorator.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	if t.accessibilityModifier.isUnderCursor(offset) {
		return t.accessibilityModifier.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if t.pattern.isUnderCursor(offset) {
		return t.pattern.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if t.ttype.isUnderCursor(offset) {
		return t.ttype.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if t.value.isUnderCursor(offset) {
		return t.value.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || t.getKind() == kind {
		return t, true
	}

	return nil, false
}

func (t *parameter) visit(exec func(nodeInterface) int) int {
	if ret := exec(t); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, decorator := range t.decorators {
		if ret := decorator.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if t.accessibilityModifier != nil {
		if ret := t.accessibilityModifier.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := t.pattern.visit(exec); ret == VisitAbort {
		return ret
	}

	if t.ttype != nil {
		if ret := t.ttype.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if t.value != nil {
		if ret := t.value.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitParameter(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	parameter := parameter{commonNode: makeCommonNode("parameter", state, node)}
	parameter.setImpl(&parameter)

	nodes := map[string]nodeInterface{}

	for index, child := range node.Children(node.Walk()) {
		commonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}

		label := node.FieldNameForChild(uint32(index))
		if label == "" {
			label = child.Kind()
		}

		nodes[label] = commonNode
	}

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

	accessibilityModifierCommonNode := nodes["accessibility_modifier"]
	var accessibility *accessibilityModifier
	if accessibilityModifierCommonNode != nil {
		if accessibilityModifier, isAccessibilityModifier := IsNode[*accessibilityModifier](accessibilityModifierCommonNode); isAccessibilityModifier {
			accessibility = accessibilityModifier
		} else {
			return nil, fmt.Errorf("invalid ast: accessibility modifier isn't an accessibilityModifier")
		}
	}

	pattern := nodes["pattern"]
	if pattern == nil {
		// No idea where "name" comes from
		pattern = nodes["name"]
	}

	if pattern == nil {
		return nil, fmt.Errorf("invalid ast: missing pattern")
	}

	parameter.accessibilityModifier = accessibility
	parameter.decorators = decorators
	parameter.isOptional = nodes["?"] != nil
	parameter.isOverride = nodes["override_modifier"] != nil
	parameter.isReadonly = nodes["readonly"] != nil
	parameter.pattern = pattern
	parameter.ttype = nodes["type"]
	parameter.value = nodes["value"]

	return &parameter, nil
}
