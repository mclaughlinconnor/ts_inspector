package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type classDeclaration struct {
	commonNode
	body           nodeInterface
	classHeritage  nodeInterface
	decorators     []*decorator
	isAbstract     bool
	name           *identifier
	typeParameters nodeInterface
}

func (a *classDeclaration) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	for _, decorator := range a.decorators {
		if decorator != nil && decorator.isUnderCursor(offset) {
			return decorator.getAstNodeOfKindAtOffset(offset, kind, first)
		}
	}

	if a.name != nil && a.name.isUnderCursor(offset) {
		return a.name.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.typeParameters != nil && a.typeParameters.isUnderCursor(offset) {
		return a.typeParameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.classHeritage != nil && a.classHeritage.isUnderCursor(offset) {
		return a.classHeritage.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.body != nil && a.body.isUnderCursor(offset) {
		return a.body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (a *classDeclaration) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	for _, decorator := range a.decorators {
		if decorator == nil {
			continue
		}

		if ret := decorator.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := a.name.visit(exec); ret == VisitAbort {
		return ret
	}

	if a.typeParameters != nil {
		if ret := a.typeParameters.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if a.classHeritage != nil {
		if ret := a.classHeritage.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := a.body.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitClassDeclaration(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	classDeclaration := classDeclaration{commonNode: makeCommonNode("classDeclaration", state, node)}
	classDeclaration.setImpl(&classDeclaration)

	decorators := []*decorator{}
	for _, decoratorNode := range node.ChildrenByFieldName("decorator", node.Walk()) {
		decoratorCommonNode, err := walk.VisitNode(&decoratorNode, state, 0, funcMap, false)
		if decoratorCommonNode == nil {
			return nil, err
		}

		var ddecorator *decorator
		if decorator, isIdentifier := isNode[*decorator](decoratorCommonNode); isIdentifier {
			ddecorator = decorator
		} else {
			return nil, fmt.Errorf("invalid ast: decorator isn't a decorator")
		}

		decorators = append(decorators, ddecorator)
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, fmt.Errorf("invalid ast: missing name")
	}

	nameCommonNode, err := walk.VisitNode(nameNode, state, 0, funcMap, false)
	if nameNode == nil {
		return nil, err
	}

	var name *identifier
	if identifier, isIdentifier := isNode[*identifier](nameCommonNode); isIdentifier {
		name = identifier
	} else {
		return nil, fmt.Errorf("invalid ast: name isn't a property identifier")
	}

	typeParametersNode := node.ChildByFieldName("type_parameters")
	var typeParameters nodeInterface
	if typeParametersNode != nil {
		typeParameters, err = walk.VisitNode(typeParametersNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	var classHeritage nodeInterface
	for _, child := range node.NamedChildren(node.Walk()) {
		if child.Kind() != "class_heritage" {
			continue
		}

		classHeritage, err = walk.VisitNode(&child, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, fmt.Errorf("invalid ast: missing body")
	}

	body, err := walk.VisitNode(bodyNode, state, 0, funcMap, false)
	if body == nil {
		return nil, err
	}

	isAbstract := false
	if prev := nameNode.PrevSibling(); prev != nil {
		if prevPrev := prev.PrevSibling(); prevPrev != nil {
			isAbstract = prevPrev.Kind() == "abstract"
		}
	}

	classDeclaration.body = body
	classDeclaration.classHeritage = classHeritage
	classDeclaration.decorators = decorators
	classDeclaration.isAbstract = isAbstract
	classDeclaration.name = name
	classDeclaration.typeParameters = typeParameters

	return &classDeclaration, nil
}
