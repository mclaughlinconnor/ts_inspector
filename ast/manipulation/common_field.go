package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type commonField struct {
	commonNode
	accessibility *accessibilityModifier
	isOptional    bool
	isOverride    bool
	isReadonly    bool
	isStatic      bool
	name          nodeInterface
}

func (a *commonField) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	if a.accessibility != nil && a.accessibility.isUnderCursor(offset) {
		return a.accessibility.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.name != nil && a.name.isUnderCursor(offset) {
		return a.name.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (a *commonField) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	if ret := a.accessibility.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := a.name.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitCommonField(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	commonField := commonField{commonNode: makeCommonNode("commonField", state, node)}
	commonField.setImpl(&commonField)

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

	accessibilityCommonNode, found := nodes["accessibility_modifier"]

	var accessibility *accessibilityModifier
	if found {
		if accessibilityModifier, isAccessibilityModifier := isNode[*accessibilityModifier](accessibilityCommonNode); isAccessibilityModifier {
			accessibility = accessibilityModifier
		} else {
			return nil, fmt.Errorf("invalid ast: accessibility isn't an accessibility")
		}
	}

	nameCommonNode, found := nodes["name"]
	if !found {
		return nil, fmt.Errorf("invalid ast: missing name")
	}

	commonField.accessibility = accessibility
	commonField.isOptional = nodes["?"] != nil
	commonField.isOverride = nodes["override_modifier"] != nil
	commonField.isReadonly = nodes["readonly"] != nil
	commonField.isStatic = nodes["static"] != nil
	commonField.name = nameCommonNode

	return &commonField, nil
}
