package parser

import (
	"ts_inspector/ast"
	"ts_inspector/ast/manipulation"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Value struct {
	ArrayValues     []*Value
	Reference       *Reference
	SpreadReference *Reference
	StringValue     string
	Type            string // "array" or "string" or "reference" or "spread"
}

type Variable struct {
	AstNode  *manipulation.AstManipulationNode
	IsExport bool
	Kind     string // const/let/var
	Name     string
	Node     *sitter.Node
	Value    *Value
}

func (v *Value) ArrayHas(c any) bool {
	for _, element := range v.ArrayValues {
		is := element.Is(c)
		if is {
			return true
		}
	}

	return false
}

func (v *Value) FlattenArray(state *State) func(func(*Reference) bool) {
	return func(yield func(*Reference) bool) {
		if v.Type != "array" {
			return
		}

		for _, element := range v.ArrayValues {
			if element.Type == "reference" {
				ref := element.Reference
				if !yield(ref) {
					return
				}
			}

			if element.Type == "spread" {
				ref := element.SpreadReference
				ref.Resolve(state)

				// Can only spread an array
				if ref.Variable == nil || ref.Variable.Value == nil || ref.Variable.Value.Type != "array" {
					continue
				}

				for e := range ref.Variable.Value.FlattenArray(state) {
					if !yield(e) {
						return
					}
				}
			}
		}
	}
}

func (v *Value) FlattenReferenceArraysToReferences(state *State) func(func(*Reference) bool) {
	return func(yield func(*Reference) bool) {
		y := func(r *Reference) bool {
			r.Resolve(state)
			return yield(r)
		}

		if v == nil {
			return
		}

		if v.Type == "reference" {
			v.Reference.Resolve(state)
			if v.Reference.Class != nil {
				if !y(v.Reference) {
					return
				}
			}

			if v.Reference.Variable != nil && v.Reference.Variable.Value != nil && v.Reference.Variable.Value.Type == "array" {
				for element := range v.Reference.Variable.Value.FlattenArray(state) {
					if !y(element) {
						return
					}
				}
			}

			if !y(v.Reference) {
				return
			}
		}

		if v.Type == "array" {
			for element := range v.FlattenArray(state) {
				if !y(element) {
					return
				}
			}
		}
	}
}

func (v *Value) Is(c any) bool {
	switch v.Type {
	case "array":
		return false
	case "spread":
		return v.SpreadReference.Class == c || v.SpreadReference.Name == c
	case "string":
		return v.StringValue == c
	case "reference":
		return v.Reference.Class == c || v.Reference.Name == c
	}

	return false
}

func (v *Value) IsOrHas(c any) bool {
	if v.Type == "array" {
		return v.ArrayHas(c)
	}

	return v.Is(c)
}

func (v *Value) Iterate(c any) bool {
	if v.Type == "array" {
		return v.ArrayHas(c)
	}

	return v.Is(c)
}

func NodeToValue(file *File, node *sitter.Node, content []byte) *Value {
	switch node.Kind() {
	case "array":
		return nodeToArrayValue(file, node, content)
	case "string":
		return &Value{StringValue: node.Utf8Text([]byte(content)), Type: "string"}
	case "spread_element":
		if node.NamedChildCount() != 1 {
			return nil
		}

		ident := node.NamedChild(0)
		if ident.Kind() != "identifier" {
			return nil
		}

		return &Value{SpreadReference: nodeToReference(file, ident, content), Type: "spread"}
	case "property_identifier":
		fallthrough
	case "identifier":
		return &Value{Reference: nodeToReference(file, node, content), Type: "reference"}
	default:
		return nil
	}
}

func AstNodeToValue(file *File, node manipulation.AstManipulationNode, content []byte) *Value {
	if node == nil {
		return nil
	}

	switch node.GetKind() {
	case "array":
		return astNodeToArrayValue(file, node, content)
	case "string":
		return &Value{StringValue: node.GetText(), Type: "string"}
	case "spreadElement":
		identifier, isIdentifier := manipulation.IsNode[*manipulation.Identifier](node)
		if !isIdentifier {
			return nil
		}

		return &Value{SpreadReference: astNodeToReference(file, identifier, content), Type: "spread"}
	case "identifier":
		identifier, isIdentifier := manipulation.IsNode[*manipulation.Identifier](node)
		if !isIdentifier {
			return nil
		}

		return &Value{Reference: astNodeToReference(file, identifier, content), Type: "reference"}
	default:
		return nil
	}
}

func nodeToArrayValue(file *File, node *sitter.Node, content []byte) *Value {
	values := make([]*Value, 0)

	for i := range node.NamedChildCount() {
		element := node.NamedChild(i)
		value := NodeToValue(file, element, content)
		if value == nil {
			continue
		}

		values = append(values, value)
	}

	return &Value{ArrayValues: values, Type: "array"}
}

func astNodeToArrayValue(file *File, node manipulation.AstManipulationNode, content []byte) *Value {
	childedNode, isChildedNodeInterface := manipulation.IsNode[manipulation.ChildedNodeInterface](node)
	if !isChildedNodeInterface {
		return nil
	}

	values := make([]*Value, 0)

	for _, element := range childedNode.GetChildren() {
		value := AstNodeToValue(file, element, content)
		if value == nil {
			continue
		}

		values = append(values, value)
	}

	return &Value{ArrayValues: values, Type: "array"}
}

func nodeToReference(file *File, node *sitter.Node, content []byte) *Reference {
	return &Reference{File: file, Name: node.Utf8Text(content), Node: node}
}

func astNodeToReference(file *File, node *manipulation.Identifier, content []byte) *Reference {
	root, err := utils.ParseText(content, utils.TypeScript)
	if err != nil {
		return nil
	}

	tsNode := ast.GetNamedNodeAtPosition(root, node.GetStartOffset())

	return &Reference{AstNode: node, File: file, Name: node.GetText(), Node: tsNode}
}
