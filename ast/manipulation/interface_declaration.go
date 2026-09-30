package manipulation

import (
	"fmt"
	"slices"
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type InterfaceDeclaration struct {
	commonNode
	Body           nodeInterface
	ExtendsClause  nodeInterface
	Name           *identifier
	TypeParameters *typeParameters
}

func (a *InterfaceDeclaration) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if !a.isUnderCursor(offset) {
		return nil, false
	}

	if first && a.getKind() == kind {
		return a, true
	}

	if a.Name != nil && a.Name.isUnderCursor(offset) {
		return a.Name.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.TypeParameters != nil && a.TypeParameters.isUnderCursor(offset) {
		return a.TypeParameters.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.ExtendsClause != nil && a.ExtendsClause.isUnderCursor(offset) {
		return a.ExtendsClause.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if a.Body != nil && a.Body.isUnderCursor(offset) {
		return a.Body.getAstNodeOfKindAtOffset(offset, kind, first)
	}

	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return nil, false
}

func (a *InterfaceDeclaration) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
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

	if a.ExtendsClause != nil {
		if ret := a.ExtendsClause.visit(exec); ret == VisitAbort {
			return ret
		}
	}

	if ret := a.Body.visit(exec); ret == VisitAbort {
		return ret
	}

	return VisitContinue
}

func visitInterfaceDeclaration(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	interfaceDeclaration := InterfaceDeclaration{commonNode: makeCommonNode("interfaceDeclaration", state, node)}
	interfaceDeclaration.setImpl(&interfaceDeclaration)

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

	var extends nodeInterface
	extendsNodeIndex := slices.IndexFunc(node.NamedChildren(node.Walk()), func(n sitter.Node) bool { return n.Kind() == "extends_type_clause" })
	if extendsNodeIndex != -1 {
		extendsNode := node.NamedChild(uint(extendsNodeIndex))

		if extendsNode != nil {
			extends, err = walk.VisitNode(extendsNode, state, 0, funcMap, false)
			if extends == nil {
				return nil, err
			}
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

	interfaceDeclaration.Body = body
	interfaceDeclaration.ExtendsClause = extends
	interfaceDeclaration.Name = name
	interfaceDeclaration.TypeParameters = ttypeParameters

	return &interfaceDeclaration, nil
}
