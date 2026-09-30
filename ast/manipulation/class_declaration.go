package manipulation

import (
	"fmt"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type ClassDeclaration struct {
	commonNode
	Body           nodeInterface
	ClassHeritage  *classHeritage
	Decorators     []*decorator
	IsAbstract     bool
	IsExport       bool
	Name           *identifier
	TypeParameters *typeParameters
}

func (a *ClassDeclaration) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
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

	if a.Name != nil && a.Name.isUnderCursor(offset) {
		return a.Name.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.TypeParameters != nil && a.TypeParameters.isUnderCursor(offset) {
		return a.TypeParameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.ClassHeritage != nil && a.ClassHeritage.isUnderCursor(offset) {
		return a.ClassHeritage.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.Body != nil && a.Body.isUnderCursor(offset) {
		return a.Body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (a *ClassDeclaration) visit(exec func(nodeInterface) int) int {
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

	if ret := a.Name.visit(exec); ret == VisitAbort {
		return ret
	}

	if a.TypeParameters != nil {
		if ret := a.TypeParameters.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if a.ClassHeritage != nil {
		if ret := a.ClassHeritage.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := a.Body.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitClassDeclaration(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	classDeclaration := ClassDeclaration{commonNode: makeCommonNode("classDeclaration", state, node)}
	classDeclaration.setImpl(&classDeclaration)

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

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, fmt.Errorf("invalid ast: missing name")
	}

	nameCommonNode, err := walk.VisitNode(nameNode, state, 0, funcMap, false)
	if nameNode == nil {
		return nil, err
	}

	var name *identifier
	if identifier, isIdentifier := IsNode[*identifier](nameCommonNode); isIdentifier {
		name = identifier
	} else {
		return nil, fmt.Errorf("invalid ast: name isn't a property identifier")
	}

	typeParametersNode := node.ChildByFieldName("type_parameters")
	var ttypeParameters *typeParameters
	if typeParametersNode != nil {
		typeParametersCommonNode, err := walk.VisitNode(typeParametersNode, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}

		if typeParameters, isTypeParameters := IsNode[*typeParameters](typeParametersCommonNode); isTypeParameters {
			ttypeParameters = typeParameters
		} else {
			return nil, fmt.Errorf("invalid ast: type parameters isn't a type parameters")
		}
	}

	var cclassHeritage *classHeritage
	for _, child := range node.NamedChildren(node.Walk()) {
		if child.Kind() != "class_heritage" {
			continue
		}

		classHeritageCommonNode, err := walk.VisitNode(&child, state, 0, funcMap, false)
		if err != nil {
			return nil, err
		}

		if classHeritage, isClassHeritage := IsNode[*classHeritage](classHeritageCommonNode); isClassHeritage {
			cclassHeritage = classHeritage
		} else {
			return nil, fmt.Errorf("invalid ast: class heritage isn't a class heritage")
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

	classDeclaration.Body = body
	classDeclaration.ClassHeritage = cclassHeritage
	classDeclaration.Decorators = decorators
	classDeclaration.IsAbstract = isAbstract
	_, classDeclaration.IsExport = IsNode[*ExportStatement](state)
	classDeclaration.Name = name
	classDeclaration.TypeParameters = ttypeParameters

	return &classDeclaration, nil
}
