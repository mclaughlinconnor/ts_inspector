package tcb_cm

import "slices"

type Scope struct {
	ParentScope *Scope
	Parts       StatementParts
	Variables   []*Variable
}

type Variable struct {
	Identifier string
	Value      string
}

func (s *Scope) AddPart(p string) {
	s.Parts.AddPart(p)
}

func (s *Scope) AddVariable(v *Variable) {
	s.Variables = append(s.Variables, v)
}

func (s *Scope) GetVariable(value StatementParts) *Variable {
	v := value.ToString()

	index := slices.IndexFunc(s.Variables, func(variable *Variable) bool { return variable.Value == v })
	if index == -1 {
		return nil
	}

	return s.Variables[index]
}
