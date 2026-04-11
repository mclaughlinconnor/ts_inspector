package tcb_cm

import (
	"slices"

	sitter "github.com/smacker/go-tree-sitter"
)

type Scope struct {
	ChildScope  *Scope
	ParentScope *Scope
	Parts       StatementParts
	Variables   []*Variable
}

type Variable struct {
	Identifier string
	Value      string
}

func (s *Scope) AddStatementParts(parts *StatementParts) {
	s.Parts.AddStatementParts(parts)
}

func (s *Scope) AddVirtPart(p string) {
	s.Parts.AddVirtPart(p)
}

func (s *Scope) AddPart(p Part) {
	s.Parts.AddPart(p)
}

func (s *Scope) AddRealPart(p string, node *sitter.Node) {
	s.Parts.AddRealPart(p, node)
}

func (s *Scope) AddVariable(v *Variable) {
	s.Variables = append(s.Variables, v)
}

func (s *Scope) GetVariableByValue(value StatementParts) *Variable {
	v := value.ToString()

	index := slices.IndexFunc(s.Variables, func(variable *Variable) bool { return variable.Value == v })
	if index == -1 {
		return nil
	}

	return s.Variables[index]
}

func (s *Scope) GetVariableByName(name string) *Variable {
	scope := s

	for scope != nil {
		index := slices.IndexFunc(s.Variables, func(variable *Variable) bool { return variable.Identifier == name })
		if index != -1 {
			return scope.Variables[index]
		}

		scope = scope.ParentScope
	}

	return nil
}
