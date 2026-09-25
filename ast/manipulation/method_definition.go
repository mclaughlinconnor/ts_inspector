package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type methodDefinition struct {
	commonNode
	accessibility  *accessibilityModifier
	body           nodeInterface
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

	body, found := nodes["body"]
	if !found {
		return nil, fmt.Errorf("invalid ast: missing body")
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

	methodDefinition.accessibility = accessibility
	methodDefinition.body = body
	methodDefinition.isAsync = nodes["async"] != nil
	methodDefinition.isGenerator = nodes["*"] != nil
	methodDefinition.isGetter = nodes["get"] != nil
	methodDefinition.isOptional = nodes["?"] != nil
	methodDefinition.isOverride = nodes["override_modifier"] != nil
	methodDefinition.isReadonly = nodes["readonly"] != nil
	methodDefinition.isSetter = nodes["set"] != nil
	methodDefinition.isStatic = nodes["static"] != nil
	methodDefinition.name = name
	methodDefinition.parameters = parameters
	methodDefinition.returnType = nodes["return_type"]
	methodDefinition.typeParameters = nodes["type_parameters"]

	return &methodDefinition, nil
}
