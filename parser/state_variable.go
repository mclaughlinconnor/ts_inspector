package parser

import sitter "github.com/smacker/go-tree-sitter"

type Value struct {
	ArrayValues     []*Value
	Reference       *Reference
	SpreadReference *Reference
	StringValue     string
	Type            string // "array" or "string" or "reference" or "spread"
}

type Variable struct {
	Kind     string // const/let/var
	IsExport bool
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
		if v == nil {
			return
		}

		if v.Type == "reference" {
			v.Reference.Resolve(state)
			if v.Reference.Class != nil {
				if !yield(v.Reference) {
					return
				}
			}

			if v.Reference.Variable != nil && v.Reference.Variable.Value != nil && v.Reference.Variable.Value.Type == "array" {
				for element := range v.Reference.Variable.Value.FlattenArray(state) {
					if !yield(element) {
						return
					}
				}
			}

			if !yield(v.Reference) {
				return
			}
		}

		if v.Type == "array" {
			for element := range v.FlattenArray(state) {
				if !yield(element) {
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

func NodeToValue(file *File, node *sitter.Node) *Value {
	switch node.Type() {
	case "array":
		return nodeToArrayValue(file, node)
	case "string":
		return &Value{StringValue: node.Content([]byte(file.Snapshot().Content)), Type: "string"}
	case "spread_element":
		if node.NamedChildCount() != 1 {
			return nil
		}

		ident := node.NamedChild(0)
		if ident.Type() != "identifier" {
			return nil
		}

		return &Value{SpreadReference: nodeToReference(file, ident), Type: "spread"}
	case "identifier":
		return &Value{Reference: nodeToReference(file, node), Type: "reference"}
	default:
		return nil
	}
}

func nodeToArrayValue(file *File, node *sitter.Node) *Value {
	values := make([]*Value, 0)

	for i := range node.NamedChildCount() {
		element := node.NamedChild(int(i))
		value := NodeToValue(file, element)
		if value == nil {
			continue
		}

		values = append(values, value)
	}

	return &Value{ArrayValues: values, Type: "array"}
}

func nodeToReference(file *File, node *sitter.Node) *Reference {
	return &Reference{File: file, Name: node.Content([]byte(file.Snapshot().Content)), Node: node}
}
