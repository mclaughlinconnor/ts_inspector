package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type methodSignature struct {
	commonNode
	accessibility  *accessibilityModifier
	isAsync        bool
	isGenerator    bool
	isGetter       bool
	isOptional     bool
	isOverride     bool
	isReadonly     bool
	isSetter       bool
	isStatic       bool
	name           *propertyIdentifier
	parameters     nodeInterface // todo: make it the formal_parameters node
	returnType     nodeInterface // type_annotation
	typeParameters nodeInterface
}

func (a *methodSignature) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
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

func (a *methodSignature) visit(exec func(nodeInterface) int) int {
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

	if a.returnType != nil {
		if ret := a.returnType.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	return VisitContinue
}

func visitMethodSignature(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	methodSignature := methodSignature{commonNode: makeCommonNode("methodSignature", state, node)}
	methodSignature.setImpl(&methodSignature)

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

	methodSignature.accessibility = accessibility
	methodSignature.isAsync = nodes["async"] != nil
	methodSignature.isGenerator = nodes["*"] != nil
	methodSignature.isGetter = nodes["get"] != nil
	methodSignature.isOptional = nodes["?"] != nil
	methodSignature.isOverride = nodes["override_modifier"] != nil
	methodSignature.isReadonly = nodes["readonly"] != nil
	methodSignature.isSetter = nodes["set"] != nil
	methodSignature.isStatic = nodes["static"] != nil
	methodSignature.name = name
	methodSignature.parameters = parameters
	methodSignature.returnType = nodes["return_type"]
	methodSignature.typeParameters = nodes["type_parameters"]

	return &methodSignature, nil
}
