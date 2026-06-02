package tcb

import (
	"slices"

	sitter "github.com/smacker/go-tree-sitter"
)

type Scope struct {
	ChildrenScopes []*Scope
	ParentScope    *Scope
	Parts          Statement
	Variables      []*Variable
}

type Variable struct {
	LastPart   *Part
	Identifier string
	RefName    string
	Value      string
}

func (s *Scope) AddStatement(parts *Statement) {
	s.Parts.AddStatement(parts)
}

func (s *Scope) AddStatementAfterPart(parts *Statement, after *Part) *Part {
	return s.Parts.AddStatementAfterPart(parts, after)
}

func (s *Scope) AddVirtPart(p string) {
	s.Parts.AddVirtPart(p)
}

func (s *Scope) AddPart(p *Part) {
	s.Parts.AddPart(p)
}

func (s *Scope) AddRealPart(p string, node *sitter.Node) {
	s.Parts.AddRealPart(p, node)
}

func (s *Scope) AddScopePart(scope *Scope) {
	s.Parts.AddScopePart(scope)
}

func (s *Scope) AddVariable(v *Variable) {
	s.Variables = append(s.Variables, v)
}

func (s *Scope) GetVariableByValue(value *Statement) *Variable {
	v := value.ToString()
	scope := s

	for scope != nil {
		index := slices.IndexFunc(scope.Variables, func(variable *Variable) bool { return variable.Value == v })
		if index != -1 {
			return scope.Variables[index]
		}

		scope = scope.ParentScope
	}

	return nil
}

func (s *Scope) GetVariableByName(name string) *Variable {
	scope := s

	for scope != nil {
		index := slices.IndexFunc(scope.Variables, func(variable *Variable) bool { return variable.Identifier == name || variable.RefName == name })
		if index != -1 {
			return scope.Variables[index]
		}

		scope = scope.ParentScope
	}

	return nil
}

func (s *Scope) ToStatement() Statement {
	statement := Statement{}

	for _, part := range s.Parts.Parts {
		if part.scope != nil {
			s := part.scope.ToStatement()
			statement.AddStatement(&s)
			continue
		}

		statement.AddPart(part)
	}

	return statement
}
