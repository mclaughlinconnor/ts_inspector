package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type methodDefinition struct {
	commonNode
	accessibility *accessibilityModifier
	body          nodeInterface
	name          *propertyIdentifier
	parameters    nodeInterface // todo: make it the formal_parameters node
	returnType    nodeInterface // type_annotation

	// isStatic       bool
	// isOverride     bool
	// isReadonly     bool
	// isAsync        bool
	// isGetter       bool
	// isSetter       bool
	// isGenerator    bool
	// isOptional     bool
	// typeParameters nodeInterface
}

func (a *methodDefinition) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	if a.body != nil && a.body.isUnderCursor(offset) {
		return a.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.parameters != nil && a.parameters.isUnderCursor(offset) {
		return a.parameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.returnType != nil && a.returnType.isUnderCursor(offset) {
		return a.returnType.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (f *methodDefinition) _buildCfgBlock() (bool, error) {
	return true, buildFunctionCfg(f, f.name.getText(), f.body)
}

func (a *methodDefinition) visit(exec func(nodeInterface) int) int {
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

	if ret := a.parameters.visit(exec); ret == VisitAbort {
		return ret
	}

	if ret := a.body.visit(exec); ret == VisitAbort {
		return ret
	}

	if a.returnType != nil {
		if ret := a.returnType.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitMethodDefinition(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	methodDefinition := methodDefinition{commonNode: makeCommonNode("methodDefinition", state, node)}
	methodDefinition.setImpl(&methodDefinition)

	nodes := map[string]nodeInterface{}

	for index, child := range node.NamedChildren(node.Walk()) {
		commonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}

		label := node.FieldNameForNamedChild(uint32(index))
		if label == "" {
			label = child.Kind()
		}

		nodes[label] = commonNode
	}

	accessibilityCommonNode, found := nodes["accessibility_modifier"]

	var accessibility *accessibilityModifier
	if found {
		if accessibilityModifier, isAccessibilityModifier := accessibilityCommonNode.isAccessibilityModifier(); isAccessibilityModifier {
			accessibility = accessibilityModifier
		} else {
			return nil, fmt.Errorf("invalid ast: accessibility isn't an accessibility")
		}
	}

	nameCommonNode, found := nodes["name"]
	if !found {
		return nil, fmt.Errorf("invalid ast: missing name")
	}

	var name *propertyIdentifier
	if propertyIdentifier, isPropertyIdentifier := nameCommonNode.isPropertyIdentifier(); isPropertyIdentifier {
		name = propertyIdentifier
	} else {
		return nil, fmt.Errorf("invalid ast: name isn't a property identifier")
	}

	parameters, found := nodes["parameters"]
	if !found {
		return nil, fmt.Errorf("invalid ast: missing parameters")
	}

	body, found := nodes["body"]
	if !found {
		return nil, fmt.Errorf("invalid ast: missing body")
	}

	returnType, found := nodes["return_type"]

	methodDefinition.accessibility = accessibility
	methodDefinition.name = name
	methodDefinition.parameters = parameters
	methodDefinition.body = body
	methodDefinition.returnType = returnType

	return &methodDefinition, nil
}
