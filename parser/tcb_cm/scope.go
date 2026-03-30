package tcb_cm

type Scope struct {
	ParentScope *Scope
	Parts StatementParts
	Variables []*any
}

func (s *Scope) AddPart(p string) {
	s.Parts.AddPart(p)
}
